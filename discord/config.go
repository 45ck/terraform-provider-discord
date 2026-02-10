package discord

import (
	"context"
	"github.com/andersfylling/disgord"
	"io"
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
	rt := lrt.Proxied
	if rt == nil {
		rt = http.DefaultTransport
	}

	// Disgord uses the provided HTTP client. This transport wrapper retries 429s defensively.
	// Important: close the body on 429 before retrying to avoid leaking connections.
	for attempt := 0; attempt < 10; attempt++ {
		res, e = rt.RoundTrip(req)
		if e != nil || res == nil {
			return res, e
		}
		if res.StatusCode != http.StatusTooManyRequests {
			return res, nil
		}

		retryAfter := res.Header.Get("X-RateLimit-Reset-After")
		if retryAfter == "" {
			retryAfter = res.Header.Get("Retry-After")
		}
		f, _ := strconv.ParseFloat(retryAfter, 64)
		if f <= 0 {
			f = 1
		}

		// Drain and close the body so the connection can be reused.
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()

		sleep := time.Duration(f*1000.0) * time.Millisecond
		if err := sleepWithContext(req.Context(), sleep); err != nil {
			return nil, err
		}
	}

	return nil, context.DeadlineExceeded
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
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
