package issues

import (
	"ActQABot/conf"
	"ActQABot/internal/grpc_utils"
	"ActQABot/pkg/github/gh_api"
	"ActQABot/pkg/hosts"
	"ActQABot/pkg/worker_report"
	"ActQABot/templates"
	"context"
	"errors"
	"fmt"
	actservice "github.com/D1-3105/ActService/api/gen/ActService"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"strings"
)

type startCallArgs struct {
	hostName     string
	commitId     string
	workflowName string `default:".github/workflows/"`
	extraFlag    []string
}

func createJob(ctx context.Context, callArgs *startCallArgs, cmd *IssuePRCommand) (*actservice.JobResponse, error) {
	hostConf, ok := conf.Hosts.Hosts[callArgs.hostName]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Unknown host %s", callArgs.hostName))
	}
	grpcConn, err := grpc_utils.NewGRPCConn(hostConf)
	if err != nil {
		glog.Errorf("unable to create connection, %s", err.Error())
		return nil, err
	}
	client := actservice.NewActServiceClient(grpcConn)
	resultExtraFlags := append([]string{}, hostConf.CustomFlags...)
	found := false
	for i, f := range resultExtraFlags {
		if strings.HasPrefix(f, "--container-options") {
			if strings.Contains(f, "=") {
				resultExtraFlags[i] = f + " " + strings.Join(callArgs.extraFlag, " ")
			} else if i+1 < len(resultExtraFlags) {
				resultExtraFlags[i+1] = resultExtraFlags[i+1] + " " + strings.Join(callArgs.extraFlag, " ")
			}
			found = true
			break
		}
	}
	if !found && len(callArgs.extraFlag) > 0 {
		resultExtraFlags = append(
			resultExtraFlags,
			"--container-options",
			strings.Join(callArgs.extraFlag, " "),
		)
	}

	job := &actservice.Job{
		RepoUrl:      fmt.Sprintf("git@github.com:%s.git", cmd.correspondingIssue.Repository.FullName),
		CommitId:     callArgs.commitId,
		WorkflowFile: &callArgs.workflowName,
		ExtraFlags:   resultExtraFlags,
	}
	glog.Infof(
		"Scheduling job of repo %s, commitId %s, workflowFile %s, extraFlags %v", job.RepoUrl, job.CommitId,
		*job.WorkflowFile, job.ExtraFlags,
	)
	actJobResponse, err := client.ScheduleActJob(ctx, job)
	if err != nil {
		glog.Errorf("unable to schedule job, %s", err.Error())
		return nil, err
	}
	return actJobResponse, nil
}

func (cmd *IssuePRCommand) startJobIssueCommentCommandExec(commandMeta *worker_report.GithubIssueMeta) (*gh_api.BotResponse, error) {
	var callArgs startCallArgs
	var err error

	if len(cmd.args) < 2 {
		return nil, errors.New("args are empty")
	}
	callArgs.hostName = cmd.args[0]
	callArgs.commitId = cmd.args[1]
	if len(cmd.args) > 2 {
		callArgs.workflowName = cmd.args[2]
	}
	if len(cmd.args) > 3 {
		callArgs.extraFlag = cmd.args[3:]
	}
	jobContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	callControl, err := hosts.HostAvbl.WrapJobCtx(callArgs.hostName, jobContext)
	if err != nil {
		return nil, err
	}
	go callControl()
	var jobResponse *actservice.JobResponse
	if conf.GeneralEnvironments.DryRunJobs {
		jobResponse, err = createJob(jobContext, &callArgs, cmd)
	} else {
		jobResponse = &actservice.JobResponse{
			JobId: uuid.NewString(),
		}
	}
	if err != nil {
		return nil, err
	}
	if commandMeta != nil {
		commandMeta.JobId = new(string)
		*commandMeta.JobId = jobResponse.JobId
		commandMeta.Host = callArgs.hostName
	}
	tmpContext := templates.NewStartCmdContext(
		cmd.history,
		callArgs.hostName,
		callArgs.extraFlag,
		jobResponse,
	)
	txt, err := tmpContext.GenText()
	if err != nil {
		glog.Errorf("failed to generate BotResponse: %v", err)
		return nil, err
	}
	return &gh_api.BotResponse{
		Owner:       cmd.correspondingIssue.Repository.Owner.Login,
		Repo:        cmd.correspondingIssue.Repository.Name,
		IssueNumber: cmd.correspondingIssue.Issue.Number,
		Text:        txt,
	}, err
}
