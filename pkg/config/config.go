package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Connection  ConnectionConfig
	Reddit      RedditConfig
	GoogleCloud GoogleCloudConfig `yaml:"google_cloud"`
	Currency    CurrencyConfig
	Functions   FunctionsConfig
}

type ConnectionConfig struct {
	Owner             string
	Admins            []string
	Server            string
	ServerName        string `yaml:"server_name"`
	Port              int
	TLS               bool
	Nick              string
	Username          string            `yaml:"user_name"`
	RealName          string            `yaml:"real_name"`
	NickServ          NickServConfig    `yaml:"nickserv"`
	ChanServ          ChanServConfig    `yaml:"chanserv"`
	PostConnect       PostConnectConfig `yaml:"post_connect"`
	NamesResponseCode string            `yaml:"names_response_code"`
}

type NickServConfig struct {
	Recipient       string
	IdentifyPattern string `yaml:"identify_pattern"`
	IdentifyCommand string `yaml:"identify_command"`
	Password        string
}

type ChanServConfig struct {
	Recipient   string
	UpCommand   string `yaml:"up_command"`
	DownCommand string `yaml:"down_command"`
}

type PostConnectConfig struct {
	Code     string
	Commands []string
	AutoJoin []string `yaml:"auto_join"`
}

type RedditConfig struct {
	UserAgent string `yaml:"user_agent"`
	Username  string
	Password  string
}

type GoogleCloudConfig struct {
	ProjectID              string `yaml:"project_id"`
	ServiceAccountFilename string `yaml:"service_account_filename"`
}

type CurrencyConfig struct {
	APIKey string `yaml:"api_key"`
}

type FunctionsConfig struct {
	Prefix           string
	EnabledFunctions map[string]FunctionConfig `yaml:"enabled"`
}

type FunctionConfig struct {
	Role                string
	ChannelStatus       string `yaml:"channel_status"`
	DenyPrivateMessages bool   `yaml:"deny_private_messages"`
	Triggers            []string
	Description         string
	Usages              []string
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
