package issues

import (
	"ActQABot/pkg/github/gh_api"
	"ActQABot/templates"
	"errors"
	"fmt"
	"strings"
)

const (
	HelpCommand string = "/help"
	StartJob    string = "/wf_start"
)

var SupportedCommands = []string{
	HelpCommand,
	StartJob,
}

type IssuePRCommand struct {
	correspondingIssue IssueComment
	command            string
	args               []string
	history            []string
}

func NewIssuePRCommand(issue IssueComment, history []string) (*IssuePRCommand, error) {
	commentHistory := strings.Split(issue.Comment.Body, templates.HistorySep)
	commentData := commentHistory[len(commentHistory)-1]
	commandData := strings.Split(commentData, " ")

	if len(commandData) == 0 {
		return nil, fmt.Errorf("comment data is empty")
	}
	if len(commandData) < 2 {
		return nil, fmt.Errorf(
			"comment text is invalid, use \n `@my_tag /supported_command args` \n Call %s to get details.",
			HelpCommand,
		)
	}

	command := commandData[1]

	return &IssuePRCommand{
		correspondingIssue: issue,
		command:            command,
		args:               commandData[2:],
		history:            history,
	}, nil
}

func (cmd *IssuePRCommand) CommandName() string {
	return cmd.command
}

func (cmd *IssuePRCommand) Exec() (*gh_api.BotResponse, error) {
	var err error = nil
	var botResponse *gh_api.BotResponse

	switch cmd.command {
	case HelpCommand:
		botResponse, err = cmd.helpIssueCommentCommandExec()
		break
	case StartJob:
		botResponse, err = cmd.startJobIssueCommentCommandExec()
		break
	default:
		return nil, errors.New("invalid command")
	}
	if err != nil {
		return nil, err
	}
	return botResponse, err
}
