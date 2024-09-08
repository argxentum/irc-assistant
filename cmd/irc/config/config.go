package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	IRC *IRC
}

type IRC struct {
	Server      string
	Port        int
	TLS         bool
	Nick        string
	QuitMessage string
	NickServ    *NickServ
	Channels    []string
}

type NickServ struct {
	IdentifyPattern string
	Password        string
}

func ReadConfig(filename string) (*Config, error) {
	f, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &Config{}

	err = yaml.Unmarshal(f, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
