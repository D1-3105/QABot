package templates

import "ActQABot/conf"

type StartJobHelpContext struct {
	StartCommand string
}

type HelpCmdContext struct {
	MultilineGithubComment
	HelpCommand       string
	SupportedCommands []string
	StartHelp         StartJobHelpContext
	Hosts             *conf.HostsEnvironment
}

func NewHelpCmdContext(
	oldText []string,
	helpCommand string,
	supportedCommands []string,
	startCtx StartJobHelpContext,
) *HelpCmdContext {
	tmpInit()
	return &HelpCmdContext{
		MultilineGithubComment: NewMultilineGithubComment(oldText, templateEnv.HelpCommandTemplate),
		HelpCommand:            helpCommand,
		SupportedCommands:      supportedCommands,
		StartHelp:              startCtx,
		Hosts:                  conf.Hosts,
	}
}

func (c *HelpCmdContext) GenText() (string, error) {
	return GenTextFromTemplate(c.tmplFile, c)
}
