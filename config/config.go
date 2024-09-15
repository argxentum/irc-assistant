package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Connection Connection
	Functions  Functions
}

type Connection struct {
	Owner       string
	Admins      []string
	Server      string
	Port        int
	TLS         bool
	Nick        string
	Username    string `yaml:"user_name"`
	RealName    string `yaml:"real_name"`
	Password    string
	NickServ    NickServ
	PostConnect PostConnect `yaml:"post_connect"`
}

type NickServ struct {
	Recipient       string
	IdentifyPattern string `yaml:"identify_pattern"`
	IdentifyCommand string `yaml:"identify_command"`
	Password        string
}

type PostConnect struct {
	Code     string
	Commands []string
	AutoJoin []string `yaml:"auto_join"`
}

type Functions struct {
	Enabled map[string]Function
}

type Function struct {
	Authorization string
	Prefix        string
	Description   string
	Usage         []string
}

func ReadConfig(filename string) (*Config, error) {
	f, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	err = yaml.Unmarshal(f, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
