package irc

import (
	"crypto/tls"
	"testing"
)

func TestNewTLSConfigVerifiesServerIdentity(t *testing.T) {
	const server = "irc.example.com"
	cfg := newTLSConfig(server)

	if cfg.InsecureSkipVerify {
		t.Fatal("TLS certificate verification is disabled")
	}
	if cfg.ServerName != server {
		t.Fatalf("ServerName = %q, want %q", cfg.ServerName, server)
	}
	if cfg.MinVersion < tls.VersionTLS12 {
		t.Fatalf("MinVersion = %d, want TLS 1.2 or newer", cfg.MinVersion)
	}
}
