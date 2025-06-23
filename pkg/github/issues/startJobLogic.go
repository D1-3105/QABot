package issues

import (
	"ActQABot/conf"
	"ActQABot/internal/grpc_utils"
	"ActQABot/pkg/github/gh_api"
	"ActQABot/pkg/hosts"
	"ActQABot/templates"
	"context"
	"errors"
	"fmt"
	actservice "github.com/D1-3105/ActService/api/gen/ActService"
	"github.com/golang/glog"
)

type startCallArgs struct {
	hostName     string
	commitId     string
	workflowName string `default:".github/workflows/"`
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
	job := &actservice.Job{
		RepoUrl:      fmt.Sprintf("https://github.com/%s.git", cmd.correspondingIssue.Repository.FullName),
		CommitId:     callArgs.commitId,
		WorkflowFile: &callArgs.workflowName,
	}
	actJobResponse, err := client.ScheduleActJob(ctx, job)
	if err != nil {
		glog.Errorf("unable to schedule job, %s", err.Error())
		return nil, err
	}
	return actJobResponse, nil
}

func (cmd *IssuePRCommand) startJobIssueCommentCommandExec() (*gh_api.BotResponse, error) {
	var callArgs startCallArgs
	if len(cmd.args) < 2 {
		return nil, errors.New("args are empty")
	}
	callArgs.hostName = cmd.args[0]
	callArgs.commitId = cmd.args[1]
	if len(cmd.args) > 2 {
		callArgs.workflowName = cmd.args[2]
	}
	jobContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	callControl, err := hosts.HostAvbl.WrapJobCtx(callArgs.hostName, jobContext)
	if err != nil {
		return nil, err
	}
	go callControl()
	jobResponse, err := createJob(jobContext, &callArgs, cmd)
	if err != nil {
		return nil, err
	}
	tmpContext := templates.NewStartCmdContext(
		cmd.history,
		callArgs.hostName,
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
