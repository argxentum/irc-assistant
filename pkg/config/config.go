package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	IRC         IRCConfig
	Web         WebConfig
	Queue       QueueConfig
	Reddit      RedditConfig
	GoogleCloud GoogleCloudConfig `yaml:"google_cloud"`
	Currency    CurrencyConfig
	Commands    CommandsConfig
	Ignore      IgnoreConfig
	Giphy       GiphyConfig
}

type IRCConfig struct {
	Owner       string
	Admins      []string
	Server      string
	ServerName  string `yaml:"server_name"`
	Port        int
	TLS         bool
	Nick        string
	Username    string            `yaml:"user_name"`
	RealName    string            `yaml:"real_name"`
	NickServ    NickServConfig    `yaml:"nickserv"`
	ChanServ    ChanServConfig    `yaml:"chanserv"`
	PostConnect PostConnectConfig `yaml:"post_connect"`
	Inactivity  InactivityConfig  `yaml:"inactivity"`
}

type WebConfig struct {
	Domain          string
	Port            int
	ExternalRootURL string `yaml:"external_root_url"`
	DefaultRedirect string `yaml:"default_redirect"`
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

type InactivityConfig struct {
	DefaultDuration string `yaml:"default_duration"`
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

type CommandsConfig struct {
	Prefix string
}

type IgnoreConfig struct {
	Users         []string
	Domains       []string
	TitlePrefixes []string `yaml:"title_prefixes"`
}

type GiphyConfig struct {
	APIKey string `yaml:"api_key"`
}

type QueueConfig struct {
	Topic        string
	Subscription string
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
