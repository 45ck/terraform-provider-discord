package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Discord API error response shape.
// Example:
//
//	{"message":"Missing Access","code":50001}
type discordAPIError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Discord rate limit response shape.
// Example:
//
//	{"message":"You are being rate limited.","retry_after":0.5,"global":false}
type discordRateLimit struct {
	Message    string  `json:"message"`
	RetryAfter float64 `json:"retry_after"`
	Global     bool    `json:"global"`
}

type DiscordHTTPError struct {
	Method     string
	Path       string
	StatusCode int
	Code       int
	Message    string
	Raw        string
}

func (e *DiscordHTTPError) Error() string {
	if e == nil {
		return "discord http error <nil>"
	}
	if e.Message != "" {
		if e.Code != 0 {
			return fmt.Sprintf("discord api error %s %s: http %d (code %d) %s", e.Method, e.Path, e.StatusCode, e.Code, e.Message)
		}
		return fmt.Sprintf("discord api error %s %s: http %d %s", e.Method, e.Path, e.StatusCode, e.Message)
	}
	if e.Raw != "" {
		return fmt.Sprintf("discord api error %s %s: http %d: %s", e.Method, e.Path, e.StatusCode, strings.TrimSpace(e.Raw))
	}
	return fmt.Sprintf("discord api error %s %s: http %d", e.Method, e.Path, e.StatusCode)
}

func IsDiscordHTTPStatus(err error, status int) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*DiscordHTTPError); ok {
		return e.StatusCode == status
	}
	return false
}

type RestClient struct {
	BaseURL   string
	Token     string
	HTTP      *http.Client
	UserAgent string

	// globalRL gates requests when Discord responds with a global rate limit.
	// Discord global limits apply across routes, so we must coordinate across concurrent requests.
	globalRL *globalRateLimiter
}

func NewRestClient(token string, httpClient *http.Client) *RestClient {
	c := &RestClient{
		BaseURL:   "https://discord.com/api/v10",
		Token:     token,
		HTTP:      httpClient,
		UserAgent: userAgent(),
		globalRL:  &globalRateLimiter{},
	}
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}
	return c
}

type globalRateLimiter struct {
	mu    sync.Mutex
	until time.Time
}

func (g *globalRateLimiter) wait(ctx context.Context) error {
	for {
		g.mu.Lock()
		until := g.until
		g.mu.Unlock()

		if until.IsZero() || time.Now().After(until) {
			return nil
		}

		d := time.Until(until)
		t := time.NewTimer(d)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
			// Loop in case another goroutine extends the window.
			continue
		}
	}
}

func (g *globalRateLimiter) setCooldown(d time.Duration) {
	// Add a small buffer to reduce flakiness on the boundary.
	if d < 0 {
		d = 0
	}
	d += 150 * time.Millisecond

	g.mu.Lock()
	defer g.mu.Unlock()

	until := time.Now().Add(d)
	if until.After(g.until) {
		g.until = until
	}
}

func (c *RestClient) DoJSON(ctx context.Context, method, path string, query url.Values, in interface{}, out interface{}) error {
	return c.doJSON(ctx, method, path, query, in, out, "")
}

// DoJSONWithReason is the same as DoJSON but sets the X-Audit-Log-Reason header when provided.
// Discord expects this header value URL-encoded.
func (c *RestClient) DoJSONWithReason(ctx context.Context, method, path string, query url.Values, in interface{}, out interface{}, reason string) error {
	return c.doJSON(ctx, method, path, query, in, out, reason)
}

func (c *RestClient) doJSON(ctx context.Context, method, path string, query url.Values, in interface{}, out interface{}, reason string) error {
	var bodyBytes []byte
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		bodyBytes = b
	}

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + path
	if query != nil {
		u.RawQuery = query.Encode()
	}

	// Retry loop for rate limits (429).
	for attempt := 0; attempt < 10; attempt++ {
		// Global limits apply across all routes; coordinate across concurrent requests.
		if c.globalRL != nil {
			if err := c.globalRL.wait(ctx); err != nil {
				return err
			}
		}

		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bot "+c.Token)
		req.Header.Set("User-Agent", c.UserAgent)
		req.Header.Set("Accept", "application/json")
		if in != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if reason != "" {
			// Must be URL-encoded; Discord decodes it for audit log entries.
			req.Header.Set("X-Audit-Log-Reason", url.QueryEscape(reason))
		}

		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}

		// Discord often returns useful JSON for errors; read it once.
		raw, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()

		if res.StatusCode == http.StatusTooManyRequests {
			var rl discordRateLimit
			if err := json.Unmarshal(raw, &rl); err != nil || rl.RetryAfter <= 0 {
				// Fallback to headers, which are seconds (may be float).
				ra := res.Header.Get("Retry-After")
				if ra == "" {
					ra = res.Header.Get("X-RateLimit-Reset-After")
				}
				if ra != "" {
					if f, ferr := strconv.ParseFloat(ra, 64); ferr == nil {
						rl.RetryAfter = f
					}
				}
				if rl.RetryAfter <= 0 {
					rl.RetryAfter = 1.0
				}
			}
			sleep := time.Duration(rl.RetryAfter*1000.0) * time.Millisecond

			// Discord can respond with a global limit; coordinate it.
			if rl.Global || strings.EqualFold(res.Header.Get("X-RateLimit-Global"), "true") {
				if c.globalRL != nil {
					c.globalRL.setCooldown(sleep)
				}
			}

			t := time.NewTimer(sleep)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
				continue
			}
		}

		if res.StatusCode < 200 || res.StatusCode > 299 {
			var apiErr discordAPIError
			if err := json.Unmarshal(raw, &apiErr); err == nil && apiErr.Message != "" {
				return &DiscordHTTPError{
					Method:     method,
					Path:       path,
					StatusCode: res.StatusCode,
					Code:       apiErr.Code,
					Message:    apiErr.Message,
				}
			}
			return &DiscordHTTPError{
				Method:     method,
				Path:       path,
				StatusCode: res.StatusCode,
				Raw:        string(raw),
			}
		}

		if out == nil {
			return nil
		}
		if len(raw) == 0 || res.StatusCode == http.StatusNoContent {
			return nil
		}
		return json.Unmarshal(raw, out)
	}

	return fmt.Errorf("discord api error %s %s: exceeded rate limit retry attempts", method, path)
}
