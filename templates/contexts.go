package templates

import (
	"ActQABot/conf"
	"bytes"
	actservice "github.com/D1-3105/ActService/api/gen/ActService"
	"github.com/golang/glog"
	"html/template"
	"path/filepath"
)

const (
	HistorySep string = "--------------"
)

var templateEnv *conf.TemplatesEnvironment

func tmpInit() {
	if templateEnv == nil {
		templateEnv = new(conf.TemplatesEnvironment)
		conf.NewEnviron(templateEnv)
	}
}

func tmpl(pth string) (func(data any) (string, error), error) {
	baseFilePath, err := filepath.Abs(templateEnv.BaseCommandTemplate)
	if err != nil {
		return nil, err
	}
	tmpFilePath, err := filepath.Abs(pth)
	if err != nil {
		return nil, err
	}
	glog.V(1).Infof("Using path %s", tmpFilePath)

	templ, err := template.ParseFiles(baseFilePath, tmpFilePath)
	if err != nil {
		return nil, err
	}

	return func(data any) (string, error) {
		var buf bytes.Buffer
		err := templ.ExecuteTemplate(&buf, "base", data)
		return buf.String(), err
	}, nil
}

func GenTextFromTemplate(tmplFile string, data any) (string, error) {
	builder, err := tmpl(tmplFile)
	if err != nil {
		return "", err
	}
	return builder(data)
}

type BaseCMD struct {
	OldText  []string
	Sep      string
	tmplFile string
}

func NewBaseCMD(oldText []string, templFile string) BaseCMD {
	tmpInit()
	return BaseCMD{
		OldText:  oldText,
		Sep:      HistorySep,
		tmplFile: templFile,
	}
}

func (c *BaseCMD) GenText() (string, error) {
	return GenTextFromTemplate(c.tmplFile, c)
}

type StartJobHelpContext struct {
	StartCommand string
}

type HelpCmdContext struct {
	BaseCMD
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
		BaseCMD:           NewBaseCMD(oldText, templateEnv.HelpCommandTemplate),
		HelpCommand:       helpCommand,
		SupportedCommands: supportedCommands,
		StartHelp:         startCtx,
		Hosts:             conf.Hosts,
	}
}

func (c *HelpCmdContext) GenText() (string, error) {
	return GenTextFromTemplate(c.tmplFile, c)
}

type StartCmdContext struct {
	BaseCMD
	JobResponse *actservice.JobResponse
	MyDSN       string
	JobHost     string
}

func NewStartCmdContext(oldText []string, jobHost string, jobResponse *actservice.JobResponse) *StartCmdContext {
	var serverEnv conf.ServerEnvironment
	conf.NewEnviron(&serverEnv)
	tmpInit()

	return &StartCmdContext{
		BaseCMD:     NewBaseCMD(oldText, templateEnv.StartCommandTemplate),
		JobResponse: jobResponse,
		MyDSN:       serverEnv.StreamDSN,
		JobHost:     jobHost,
	}
}

func (c *StartCmdContext) GenText() (string, error) {
	return GenTextFromTemplate(c.tmplFile, c)
}
