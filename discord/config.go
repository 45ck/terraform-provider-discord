package discord

import (
	"net/http"
)

type Config struct {
	Token    string
	ClientID string
	Secret   string
}

type Context struct {
	Rest   *RestClient
	Config *Config
}

func (c *Config) Client() (*Context, error) {
	httpClient := &http.Client{Transport: http.DefaultTransport}
	return &Context{
		Rest:   NewRestClient(c.Token, httpClient),
		Config: c,
	}, nil
}
