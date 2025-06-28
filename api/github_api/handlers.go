package github_api

import (
	"ActQABot/conf"
	"ActQABot/internal/grpc_utils"
	"ActQABot/pkg/github/gh_api"
	"ActQABot/pkg/github/issues"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	actservice "github.com/D1-3105/ActService/api/gen/ActService"
	"github.com/golang/glog"
	"github.com/gorilla/schema"
	"io"
	"net/http"
	"time"
)

func returnError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"error": "%s!"}`, err.Error())))
}

func returnErrorEvent(w http.ResponseWriter, err error) {
	_, _ = w.Write([]byte(fmt.Sprintf(`data: {"error": "%s!"}`, err.Error())))
}

// webhookHandler handles incoming GitHub webhook events.
// @Summary GitHub webhook
// @Description GitHub Webhooks: issue_comment, ping etc.
// @Tags github
// @Accept json
// @Produce json
// @Param X-GitHub-Event header string true "GitHub Event Type (e.g. 'issue_comment')"
// @Param WebhookQuery query github_api.WebhookQuery true "Query parameters"
// @Param payload body github_api.IssueCommentEvent true "Webhook payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /github/events/ [post]
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	eventType := r.Header.Get("X-GitHub-Event")
	decoderSchema := schema.NewDecoder()
	q := WebhookQuery{
		PostBack: true,
	}
	err := decoderSchema.Decode(&q, r.URL.Query())
	if err != nil {
		returnError(w, err)
		return
	}
	decoder := json.NewDecoder(r.Body)
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(r.Body)

	switch eventType {
	case "ping":
		break
	case "issue_comment":
		var issue IssueCommentEvent
		if err := decoder.Decode(&issue); err != nil {
			returnError(w, err)
			return
		}
		if err := issueHandler(&issue, q.PostBack); err != nil {
			returnError(w, err)
			return
		}

		break
	case "pull_request":
		break
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"event_type": "%s"}`, eventType) + "\n"))
}

func issueHandler(issueComment *IssueCommentEvent, postBack bool) error {
	var resp *gh_api.BotResponse
	if issueComment.Action == "created" {
		issueCommand, err := issues.NewIssuePRCommand(issueComment.IssueComment, []string{})
		if err != nil {
			glog.Errorf("NewIssuePRCommand error: %v", err)
			return err
		}
		resp, err = issueCommand.Exec()
		if err != nil {
			glog.Errorf("issueCommand.Exec error: %v", err)
			return err
		}
	}
	if resp != nil {
		if postBack {
			go func() {
				err := gh_api.PostIssueCommentFunc(resp, conf.GithubEnvironment.Token)
				if err != nil {
					glog.Errorf("PostIssueCommentFunc error: %v", err)
				}
			}()
		} else {
			glog.Infof("IssuePR response \n\n%s\n", resp.Text)
		}
	}
	return nil
}

// logStreamer streams logs over Server-Sent Events (SSE).
// @Summary Stream job logs
// @Description Stream logs from a remote job using gRPC and send via SSE
// @Tags logs
// @Produce text/event-stream
// @Param LogStreamQuery query github_api.LogStreamQuery true "Query parameters"
// @Success 200 {string} string "data: ..."
// @Failure 400 {object} map[string]string
// @Router /job/logs/ [get]
func logStreamer(w http.ResponseWriter, r *http.Request) {
	// return option
	if r.Method == http.MethodOptions {
		glog.Info("Options OK")
		w.WriteHeader(http.StatusOK)
		return
	}
	//
	var decoder = schema.NewDecoder()
	var q LogStreamQuery
	err := decoder.Decode(&q, r.URL.Query())
	if err != nil {
		returnError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		returnError(w, fmt.Errorf("streaming is not supported"))
		return
	}

	host, ok := conf.Hosts.Hosts[q.Host]
	if !ok {
		returnError(w, fmt.Errorf("host not found"))
		return
	}
	grpcConn, err := grpc_utils.NewGRPCConn(host)
	if err != nil {
		glog.Errorf("grpc_utils.NewGRPCConn error: %v; %v", err, q)
		returnError(w, errors.New("this host is inaccessible! can't listen to his jobs"))
		return
	}
	grpcClient := actservice.NewActServiceClient(grpcConn)
	streamLogRequest := actservice.JobLogRequest{JobId: q.JobId, LastOffset: 0}
	stream, err := grpcClient.JobLogStream(r.Context(), &streamLogRequest)
	streamQ := make(chan *actservice.JobLogMessage)
	streamErrChan := make(chan error)

	streamContext, cancelStream := context.WithCancel(context.Background())
	go func() {
		defer close(streamQ)
		defer close(streamErrChan)

		defer cancelStream()
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				glog.Infof("stream %v: EOF", q)
				return
			} else if err != nil {
				streamErrChan <- err
			} else {
				streamQ <- msg
			}
		}
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case err = <-streamErrChan:
			if err != nil {
				glog.Errorf("stream %v: %v", q, err)
				returnErrorEvent(w, err)
				return
			}
		case <-streamContext.Done():
			glog.Infof("stream %v: context done", q)
			return
		case msg := <-streamQ:
			if msg == nil {
				break
			}
			jsonedData, err := json.Marshal(msg)
			if err != nil {
				glog.Errorf("json.Marshal error: %v; %v", err, q)
				returnErrorEvent(w, errors.New("failed to unmarshal upstream message"))
				return
			}
			if _, err = fmt.Fprintf(w, "data: %s\n", jsonedData); err != nil {
				glog.Errorf("write error: %v; %v", err, q)
				return
			}
		case <-r.Context().Done():
			glog.Errorf("stream %v: client disconnected", q)
			return
		case <-ticker.C:
			_, err := fmt.Fprintf(w, "event: %d\n", time.Now().UnixMilli())
			if err != nil {
				return
			}
			flusher.Flush()
		case <-time.After(time.Minute * 10):
			glog.Errorf("streaming timeout for %v", q)
			return
		}
	}
}

// helpCommand returns md content.
// @Summary Help analog
// @Description Returns md
// @Tags command
// @Produce application/json
// @Success 200 {object} HelpCommandResponse
// @Failure 400 {object} map[string]string
// @Router /help [get]
func helpCommand(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	issueComment := IssueCommentEvent{
		issues.IssueComment{
			Comment: struct {
				Body string `json:"body"`
				User struct {
					Login string `json:"login"`
				} `json:"user"`
			}{Body: fmt.Sprintf("@ci_bot %s", issues.HelpCommand), User: struct {
				Login string `json:"login"`
			}{Login: "any"}},
		},
	}
	command, err := issues.NewIssuePRCommand(issueComment.IssueComment, []string{})
	if err != nil {
		returnError(w, err)
		return
	}
	execed, err := command.Exec()
	if err != nil {
		returnError(w, err)
		return
	}
	result := HelpCommandResponse{
		Body: execed.Text,
	}
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		returnError(w, err)
		return
	}
}
