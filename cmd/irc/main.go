package main

import (
	"crypto/tls"
	"fmt"
	"franklin/cmd/irc/config"
	irc "github.com/thoj/go-ircevent"
	"strings"
)

func main() {
	cfg, err := config.ReadConfig("config.yaml")
	if err != nil {
		panic(err)
	}

	var conn *irc.Connection
	connect(cfg, conn)
}

func connect(cfg *config.Config, conn *irc.Connection) {
	conn = irc.IRC(cfg.IRC.Nick, cfg.IRC.Nick)
	conn.Debug = true
	conn.VerboseCallbackHandler = true
	conn.QuitMessage = cfg.IRC.QuitMessage

	if cfg.IRC.TLS {
		conn.UseTLS = cfg.IRC.TLS
		conn.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	conn.AddCallback("NOTICE", func(event *irc.Event) {
		if strings.Contains(event.Message(), cfg.IRC.NickServ.IdentifyPattern) {
			conn.Privmsgf("NickServ", "IDENTIFY %s", cfg.IRC.NickServ.Password)
		}
	})

	conn.AddCallback("900", func(event *irc.Event) {
		for _, channel := range cfg.IRC.Channels {
			conn.Join(channel)
		}
	})

	conn.AddCallback("JOIN", func(event *irc.Event) {
		channel := event.Message()
		conn.Privmsgf(channel, fmt.Sprintf("Hello, %s!", channel))
	})

	_ = conn.Connect(fmt.Sprintf("%s:%d", cfg.IRC.Server, cfg.IRC.Port))
	conn.Loop()
}
