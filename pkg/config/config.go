package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IRC            IRCConfig
	Web            WebConfig
	Queue          QueueConfig
	Reddit         RedditConfig
	GoogleCloud    GoogleCloudConfig `yaml:"google_cloud"`
	Currency       APIKeyConfig
	Commands       CommandsConfig
	Ignore         IgnoreConfig
	DisinfoPenalty DisinfoPenaltyConfig `yaml:"disinfo_penalty"`
	Giphy          APIKeyConfig
	Imgflip        ImgflipConfig
	MerriamWebster MerriamWebsterConfig `yaml:"merriam_webster"`
	Firecrawl      FirecrawlConfig
	Alphavantage   APIKeyConfig
	Finage         APIKeyConfig
	Finnhub        APIKeyConfig
	Polygon        APIKeyConfig
	MarketData     APIKeyConfig `yaml:"market_data"`
	Summary        SummaryConfig
}

type IRCConfig struct {
	Owner          string
	Admins         []string
	Server         string
	ServerName     string `yaml:"server_name"`
	Port           int
	TLS            bool
	Nick           string
	Username       string            `yaml:"user_name"`
	RealName       string            `yaml:"real_name"`
	ReconnectDelay int               `yaml:"reconnect_delay"`
	NickServ       NickServConfig    `yaml:"nickserv"`
	ChanServ       ChanServConfig    `yaml:"chanserv"`
	PostConnect    PostConnectConfig `yaml:"post_connect"`
	Inactivity     InactivityConfig  `yaml:"inactivity"`
}

func (c IRCConfig) IsOwnerOrAdmin(nick string) bool {
	if c.Owner == nick {
		return true
	}

	for _, admin := range c.Admins {
		if admin == nick {
			return true
		}
	}

	return false
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
	RecoverCommand  string `yaml:"recover_command"`
	ReleaseCommand  string `yaml:"release_command"`
	Password        string
}

type ChanServConfig struct {
	Recipient   string
	UpCommand   string `yaml:"up_command"`
	DownCommand string `yaml:"down_command"`
}

type PostConnectConfig struct {
	Code      string
	Commands  []string
	AutoJoin  []string `yaml:"auto_join"`
	AutoLeave []string `yaml:"auto_leave"`
}

const InactivityModelReddit = "reddit"
const InactivityModelDrudge = "drudge"

type InactivityConfig struct {
	DefaultDuration string `yaml:"default_duration"`
	Model           string `yaml:"model"`
	Subreddit       string `yaml:"subreddit"`
	Category        string `yaml:"category"`
	Posts           int    `yaml:"posts"`
}

type RedditConfig struct {
	UserAgent               string `yaml:"user_agent"`
	Username                string
	Password                string
	ClientID                string   `yaml:"client_id"`
	ClientSecret            string   `yaml:"client_secret"`
	SummarizationSubreddits []string `yaml:"summarization_subreddits"`
}

type GoogleCloudConfig struct {
	ProjectID              string `yaml:"project_id"`
	ServiceAccountFilename string `yaml:"service_account_filename"`
	MappingAPIKey          string `yaml:"mapping_api_key"`
}

type APIKeyConfig struct {
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

type DisinfoPenaltyConfig struct {
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	Threshold      int    `yaml:"threshold"`
	Duration       string `yaml:"duration"`
}

type ImgflipConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type QueueConfig struct {
	Topic        string
	Subscription string
}

type MerriamWebsterConfig struct {
	DictionaryAPIKey string `yaml:"dictionary_api_key"`
	ThesaurusAPIKey  string `yaml:"thesaurus_api_key"`
}

type FirecrawlConfig struct {
	APIKey string `yaml:"api_key"`
}

type SummaryConfig struct {
	AvoidanceDomains  map[string]string `yaml:"avoidance_domains"`
	TranslatedDomains map[string]string `yaml:"translated_domains"`
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
