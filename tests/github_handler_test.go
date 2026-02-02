package tests

import (
	"ActQABot/api/github_api"
	"ActQABot/pkg/github/issues"
	"ActQABot/pkg/worker_report"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"
)

func TestWebhookHandler_IssueCommentCreated_Help(t *testing.T) {
	setupTestEnv(t)
	commentPosted := postIssueCommentFixture(t)
	payload := issueCommentPayload{
		Action: "created",
		IssueComment: mockComment{
			Body: fmt.Sprintf("@bot %s", issues.HelpCommand),
			User: struct {
				Login string `json:"login"`
			}{
				Login: "test-user",
			},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/github/events/", bytes.NewBuffer(body))
	req.Header.Set("X-GitHub-Event", "issue_comment")
	w := httptest.NewRecorder()

	router := github_api.Router()
	router.ServeHTTP(w, req)

	resp := w.Result()

	var content []byte

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	_, err := resp.Body.Read(content)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d: %s", resp.StatusCode, string(content))
	}

	select {
	case botResp := <-commentPosted:
		t.Logf("comment posted \n%s", botResp.Text)
	case <-time.After(time.Second * 2):
		t.Fatal("comment posted timeout")
	}
}

func TestWebhookHandler_IssueCommentCreated_StartJob(t *testing.T) {
	setupTestEnv(t)
	mocked := mockGithubMetaEtcd(mocksForGithubMetaEtcd{})
	commentPosted := postIssueCommentFixture(t)
	grpcConnFixture(t)
	payload := issueCommentPayload{
		Action: "created",
		IssueComment: mockComment{
			Body: fmt.Sprintf(
				"@bot %s my-vm some-commit .github/workflows/whatever.yml -e TEST_CASE=data\n\n",
				issues.StartJob,
			),
			User: struct {
				Login string `json:"login"`
			}{
				Login: "test-user",
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/github/events/", bytes.NewBuffer(body))
	req.Header.Set("X-GitHub-Event", "issue_comment")
	w := httptest.NewRecorder()

	router := github_api.Router()
	router.ServeHTTP(w, req)

	resp := w.Result()

	var content []byte

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	_, err := resp.Body.Read(content)

	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d: %s", resp.StatusCode, string(content))
	}

	select {
	case botResp := <-commentPosted:
		t.Logf("comment posted \n%s", botResp.Text)
		searchPat, err := regexp.Compile("job_id=(.*)")
		if err != nil {
			panic(err)
		}
		searchRes := searchPat.FindStringSubmatch(botResp.Text)
		jobId := searchRes[1]
		if jobId == "" {
			t.Fatal("job_id not found in comment")
		}
		require.Condition(
			t,
			func() bool {
				ser, ok := mocked.jobs[jobId]
				if !ok {
					return false
				}
				var deser worker_report.GithubIssueMeta
				err := json.Unmarshal(ser, &deser)
				if err != nil {
					panic(err)
				}
				t.Logf("deserialized: %s", spew.Sdump(deser))
				return deser.Body == payload.IssueComment.Body &&
					deser.Host == "my-vm" &&
					deser.Sender == "test-user" &&
					*deser.JobId == jobId &&
					*deser.MyLeaseID == *mocked.lastLease
			},
		)
	case <-time.After(time.Second * 2):
		t.Fatal("comment posted timeout")
	}

}

func TestWebhookHandler_IssueCommentCreated_StartJob_Error(t *testing.T) {
	setupTestEnv(t)
	mockGithubMetaEtcd(mocksForGithubMetaEtcd{}) // void
	commentPosted := postIssueCommentFixture(t)
	grpcConnFixture(t)
	payload := issueCommentPayload{
		Action: "created",
		IssueComment: mockComment{
			Body: fmt.Sprintf("@bot %s my-vm-error .github/workflows/", issues.StartJob),
			User: struct {
				Login string `json:"login"`
			}{
				Login: "test-user",
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/github/events/", bytes.NewBuffer(body))
	req.Header.Set("X-GitHub-Event", "issue_comment")
	w := httptest.NewRecorder()

	router := github_api.Router()
	router.ServeHTTP(w, req)

	resp := w.Result()

	var content []byte

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	_, err := resp.Body.Read(content)

	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d: %s", resp.StatusCode, string(content))
	}

	select {
	case botResp := <-commentPosted:
		t.Logf("comment posted \n%s", botResp.Text)
	case <-time.After(time.Second * 2):
		t.Fatal("comment posted timeout")
	}
}

func TestWebhookHandler_IssueCommentCreated_StartJob_NoError(t *testing.T) {
	setupTestEnv(t)
	mockGithubMetaEtcd(mocksForGithubMetaEtcd{}) // void

	commentPosted := postIssueCommentFixture(t)
	grpcConnFixture(t)
	payload := issueCommentPayload{
		Action: "created",
		IssueComment: mockComment{
			Body: fmt.Sprintf("@not-bot %s my-vm-error .github/workflows/", issues.StartJob),
			User: struct {
				Login string `json:"login"`
			}{
				Login: "test-user",
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/github/events/", bytes.NewBuffer(body))
	req.Header.Set("X-GitHub-Event", "issue_comment")
	w := httptest.NewRecorder()

	router := github_api.Router()
	router.ServeHTTP(w, req)

	resp := w.Result()

	var content []byte

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	_, err := resp.Body.Read(content)

	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d: %s", resp.StatusCode, string(content))
	}

	select {
	case botResp := <-commentPosted:
		t.Logf("comment posted \n%s", botResp.Text)
		t.Fatal("Bot replied to a non-bot comment")
	case <-time.After(time.Second * 2):
		t.Log("Bot didn't reply to non-bot comment")
	}
}

func TestWebhookHandler_IssueCommentCreated_Empty(t *testing.T) {
	setupTestEnv(t)
	commentPosted := postIssueCommentFixture(t)
	grpcConnFixture(t)
	payload := issueCommentPayload{
		Action: "created",
		IssueComment: mockComment{
			Body: fmt.Sprint(""),
			User: struct {
				Login string `json:"login"`
			}{
				Login: "test-user",
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/github/events/", bytes.NewBuffer(body))
	req.Header.Set("X-GitHub-Event", "issue_comment")
	w := httptest.NewRecorder()

	router := github_api.Router()
	router.ServeHTTP(w, req)

	resp := w.Result()

	var content []byte

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	_, err := resp.Body.Read(content)

	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d: %s", resp.StatusCode, string(content))
	}

	select {
	case botResp := <-commentPosted:
		t.Logf("comment posted \n%s", botResp.Text)
		t.Fatal("Bot replied to a non-bot comment")
	case <-time.After(time.Second * 2):
		t.Log("Bot didn't reply to non-bot comment")
	}
}
