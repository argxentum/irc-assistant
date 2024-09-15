package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Connection ConnectionConfig
	Functions  FunctionsConfig
}

type ConnectionConfig struct {
	Owner             string
	Admins            []string
	Server            string
	Port              int
	TLS               bool
	Nick              string
	Username          string `yaml:"user_name"`
	RealName          string `yaml:"real_name"`
	Password          string
	NickServ          NickServConfig
	PostConnect       PostConnectConfig `yaml:"post_connect"`
	NamesResponseCode string            `yaml:"names_response_code"`
}

type NickServConfig struct {
	Recipient       string
	IdentifyPattern string `yaml:"identify_pattern"`
	IdentifyCommand string `yaml:"identify_command"`
	Password        string
}

type PostConnectConfig struct {
	Code     string
	Commands []string
	AutoJoin []string `yaml:"auto_join"`
}

type FunctionsConfig struct {
	Prefix           string
	EnabledFunctions map[string]FunctionConfig `yaml:"enabled"`
}

type FunctionConfig struct {
	Authorization     string
	AllowedUserStatus string `yaml:"allowed_user_status"`
	Triggers          []string
	Description       string
	Usages            []string
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
