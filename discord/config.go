package discord

import (
	"github.com/andersfylling/disgord"
	"net/http"
	"strconv"
	"time"
)

type Config struct {
	Token    string
	ClientID string
	Secret   string
}

type Context struct {
	Client *disgord.Client
	Rest   *RestClient
	Config *Config
}

// This type implements the http.RoundTripper interface
type LimitedRoundTripper struct {
	Proxied http.RoundTripper
}

func (lrt LimitedRoundTripper) RoundTrip(req *http.Request) (res *http.Response, e error) {
	// Send the request, get the response (or the error)
	res, e = lrt.Proxied.RoundTrip(req)

	if res != nil && res.StatusCode == 429 {
		retryAfter := res.Header.Get("X-RateLimit-Reset-After")
		if retryAfter == "" {
			retryAfter = res.Header.Get("Retry-After")
		}

		// Discord headers are seconds (may be float). Be conservative.
		f, _ := strconv.ParseFloat(retryAfter, 64)
		if f <= 0 {
			f = 1
		}
		time.Sleep(time.Duration(f*1000.0) * time.Millisecond)

		return lrt.RoundTrip(req)
	}

	return
}

func (c *Config) Client() (*Context, error) {
	httpClient := &http.Client{Transport: LimitedRoundTripper{http.DefaultTransport}}
	client := disgord.New(disgord.Config{
		BotToken:   c.Token,
		HTTPClient: httpClient,
	})

	return &Context{
		Client: client,
		Rest:   NewRestClient(c.Token, httpClient),
		Config: c,
	}, nil
}
