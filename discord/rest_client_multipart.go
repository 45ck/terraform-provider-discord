package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (c *RestClient) DoMultipartWithReason(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	fields map[string]string,
	fileField string,
	fileName string,
	fileBytes []byte,
	out interface{},
	reason string,
) error {
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

	// Rate-limit retry loop.
	for attempt := 0; attempt < 10; attempt++ {
		// Coordinate global limits across concurrent requests.
		if c.globalRL != nil {
			if err := c.globalRL.wait(ctx); err != nil {
				return err
			}
		}

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)

		for k, v := range fields {
			_ = w.WriteField(k, v)
		}
		if fileField != "" {
			fw, err := w.CreateFormFile(fileField, fileName)
			if err != nil {
				_ = w.Close()
				return err
			}
			if _, err := fw.Write(fileBytes); err != nil {
				_ = w.Close()
				return err
			}
		}
		_ = w.Close()

		req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(buf.Bytes()))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bot "+c.Token)
		req.Header.Set("User-Agent", c.UserAgent)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", w.FormDataContentType())
		if reason != "" {
			req.Header.Set("X-Audit-Log-Reason", url.QueryEscape(reason))
		}

		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}

		raw, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()

		if res.StatusCode == http.StatusTooManyRequests {
			var rl discordRateLimit
			if err := json.Unmarshal(raw, &rl); err != nil || rl.RetryAfter <= 0 {
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

	return &DiscordHTTPError{
		Method:     method,
		Path:       path,
		StatusCode: 429,
		Message:    "exceeded rate limit retry attempts",
	}
}
