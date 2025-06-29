package conf

import (
	"github.com/caarlos0/env/v11"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"os"
)

var GeneralEnvironments GeneralEnvironment
var GithubEnvironment GithubAPIEnvironment
var Hosts *HostsEnvironment

//

type Host struct {
	Address        string   `yaml:"address"`
	MaxConcurrency int      `yaml:"max_concurrent_jobs"`
	TlsCert        *string  `yaml:"tls_cert"`
	CustomFlags    []string `yaml:"custom_flags"`
}

type HostsEnvironment struct {
	Hosts map[string]Host
}

//

type ServerEnvironment struct {
	Address        string `env:"SERVER_ADDRESS" envDefault:":8080"`
	StreamDSN      string `env:"STREAM_DSN" envDefault:"http://localhost:8000"`
	AllowOrigins   string `env:"ALLOW_ORIGINS" envDefault:"*"`
	StaticFileRoot string `env:"STATIC_FILE_ROOT"`
}

type GeneralEnvironment struct {
	HostConf string `env:"HOST_CONF"`
}

type GithubAPIEnvironment struct {
	AppID          string `env:"GITHUB_APP_ID"`
	PrivateKeyPath string `env:"GITHUB_PRIVATE_KEY_PATH"`
}

//

type TemplatesEnvironment struct {
	HelpCommandTemplate  string `env:"HELP_TEMPLATE" envDefault:"assets/help.tpl"`
	StartCommandTemplate string `env:"START_TEMPLATE" envDefault:"assets/start.tpl"`
	BaseCommandTemplate  string `env:"BASE_TEMPLATE" envDefault:"assets/base.tpl"`
}

func NewEnviron(environ any) {
	if err := env.Parse(environ); err != nil {
		panic(err)
	}
}

func NewHostsEnvironment(hostsConf string) (*HostsEnvironment, error) {
	var hosts HostsEnvironment
	hostsConfFile, err := os.Open(hostsConf)
	if err != nil {
		glog.Errorf("failed to open hosts configuration file: %s", err)
		return nil, err
	}
	err = yaml.NewDecoder(hostsConfFile).Decode(
		&hosts,
	)
	if err != nil {
		return nil, err
	}
	return &hosts, nil
}
