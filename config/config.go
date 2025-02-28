package config

import (
	"crypto/rand"
	"slices"

	"github.com/jesseduffield/generics/slices"
)

type Config struct {
	ID                 string
	ChatPort           string
	ListenPort         string
	PrivateAuthKeyFile string
	PublickAuthKeyFile string
}

var (
	defaultID                 = rand.Text() + rand.Text()
	defaultChatPort           = ":9000"
	defaultPrivateAuthKeyFile = "ecdsa_private.pem"
	defaultPublicAuthKeyFile  = "ecdsa_public.pem"
)

type WithFn func(c Config) Config

func WithID(v string) WithFn {
	return func(c Config) Config {
		c.ID = v
		return c
	}
}

func WithChatPort(v string) WithFn {
	return func(c Config) Config {
		c.ChatPort = v
		return c
	}
}

func WithListenPort(v string) WithFn {
	return func(c Config) Config {
		c.ListenPort = v
		return c
	}
}

func WithPrivateAuthKeyFile(v string) WithFn {
	return func(c Config) Config {
		c.PrivateAuthKeyFile = v
		return c
	}
}

func WithPublicAuthKeyFile(v string) WithFn {
	return func(c Config) Config {
		c.PublickAuthKeyFile = v
		return c
	}
}

func NewConfig(with ...WithFn) Config {
	conf := Config{
		ID:                 defaultID,
		ChatPort:           defaultChatPort,
		ListenPort:         "",
		PrivateAuthKeyFile: defaultPrivateAuthKeyFile,
		PublickAuthKeyFile: defaultPublicAuthKeyFile,
	}

	for _, fn := range with {
		conf = fn(conf)
	}

	return conf
}
