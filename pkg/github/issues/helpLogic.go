package issues

import (
	"ActQABot/pkg/github/gh_api"
	"ActQABot/templates"
)

func (cmd *IssuePRCommand) helpIssueCommentCommandExec() (*gh_api.BotResponse, error) {
	helpCmd := templates.NewHelpCmdContext(
		cmd.history,
		HelpCommand,
		SupportedCommands,
		templates.StartJobHelpContext{StartCommand: StartJob},
	)
	txt, err := helpCmd.GenText()
	return &gh_api.BotResponse{
		Owner:       cmd.correspondingIssue.Repository.Owner.Login,
		Repo:        cmd.correspondingIssue.Repository.Name,
		IssueNumber: cmd.correspondingIssue.Issue.Number,
		Text:        txt,
	}, err
}
