package templates

import (
	"ActQABot/conf"
	"github.com/D1-3105/ActService/api/gen/ActService"
)

type StartCmdContext struct {
	MultilineGithubComment
	JobResponse *actservice.JobResponse
	MyDSN       string
	JobHost     string
	CustomFlags []string
}

func NewStartCmdContext(
	oldText []string, jobHost string,
	customFlags []string, jobResponse *actservice.JobResponse,
) *StartCmdContext {
	var serverEnv conf.ServerEnvironment
	conf.NewEnviron(&serverEnv)
	tmpInit()

	return &StartCmdContext{
		MultilineGithubComment: NewMultilineGithubComment(oldText, templateEnv.StartCommandTemplate),
		JobResponse:            jobResponse,
		MyDSN:                  serverEnv.StreamDSN,
		JobHost:                jobHost,
		CustomFlags:            customFlags,
	}
}

func (c *StartCmdContext) GenText() (string, error) {
	return GenTextFromTemplate(c.tmplFile, c)
}
