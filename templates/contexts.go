package templates

import (
	"ActQABot/conf"
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
