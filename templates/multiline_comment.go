package templates

import (
	"bytes"
	"github.com/golang/glog"
	"path/filepath"
	"strings"
	"text/template"
)

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

	funcMap := template.FuncMap{
		"splitLines": func(s string) []string {
			return strings.Split(s, "\n")
		},
	}

	templ := template.New("base").Funcs(funcMap)

	templ, err = templ.ParseFiles(baseFilePath, tmpFilePath)
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

type MultilineGithubComment struct {
	OldText  []string
	Sep      string
	tmplFile string
}

func NewMultilineGithubComment(oldText []string, templFile string) MultilineGithubComment {
	tmpInit()
	return MultilineGithubComment{
		OldText:  oldText,
		Sep:      HistorySep,
		tmplFile: templFile,
	}
}

func (c *MultilineGithubComment) GenText() (string, error) {
	return GenTextFromTemplate(c.tmplFile, c)
}
