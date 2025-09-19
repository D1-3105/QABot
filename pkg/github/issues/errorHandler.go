package issues

import (
	"ActQABot/pkg/github/gh_api"
	"ActQABot/templates"
	"github.com/golang/glog"
)

func ErrorToBotResponse(err error, incomingIssue *IssueComment) *gh_api.BotResponse {
	if err != nil {
		errCtx := templates.NewErrorResultContext(err.Error())
		resp := &gh_api.BotResponse{
			Text:        "",
			Owner:       incomingIssue.Repository.Owner.Login,
			Repo:        incomingIssue.Repository.Name,
			IssueNumber: incomingIssue.Issue.Number,
		}
		errText, err2 := errCtx.GenText()
		if err2 != nil {
			glog.Errorf("Error generating error response: %v", err2)
			resp.Text = "Error generating error response: " + err2.Error() + ".\n"
		} else {
			resp.Text = errText + "\n"
		}
		return resp
	}
	return nil
}
