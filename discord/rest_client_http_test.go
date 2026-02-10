package discord

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"sync/atomic"
)

func TestRestClient_DoJSONWithReason_SetsAuditLogReasonHeader(t *testing.T) {
	t.Parallel()

	reason := "hello world / 123"
	want := url.QueryEscape(reason)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Audit-Log-Reason")
		if got != want {
			t.Fatalf("X-Audit-Log-Reason header mismatch: got %q want %q", got, want)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer s.Close()

	c := NewRestClient("TOKEN", s.Client())
	c.BaseURL = s.URL

	var out map[string]interface{}
	if err := c.DoJSONWithReason(context.Background(), "PATCH", "/x", nil, map[string]interface{}{"a": 1}, &out, reason); err != nil {
		t.Fatalf("DoJSONWithReason returned error: %v", err)
	}
	if v, ok := out["ok"].(bool); !ok || !v {
		t.Fatalf("unexpected response: %#v", out)
	}
}

func TestRestClient_DoJSON_RetriesOn429(t *testing.T) {
	t.Parallel()

	var calls int
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			// Keep retry small so test is fast.
			_, _ = io.WriteString(w, `{"message":"You are being rate limited.","retry_after":0.01,"global":false}`)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{"calls": calls})
	}))
	defer s.Close()

	c := NewRestClient("TOKEN", s.Client())
	c.BaseURL = s.URL

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var out map[string]interface{}
	if err := c.DoJSON(ctx, "GET", "/x", nil, nil, &out); err != nil {
		t.Fatalf("DoJSON returned error: %v", err)
	}
	if calls < 2 {
		t.Fatalf("expected at least 2 calls due to retry, got %d", calls)
	}
}

func TestRestClient_DoJSON_GlobalRateLimitBlocksOtherRequests(t *testing.T) {
	t.Parallel()

	var first429 int32
	served429 := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First request to /global returns a global 429 once.
		if r.URL.Path == "/global" && atomic.CompareAndSwapInt32(&first429, 0, 1) {
			w.Header().Set("X-RateLimit-Global", "true")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"message":"You are being rate limited.","retry_after":0.25,"global":true}`)
			close(served429)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer s.Close()

	c := NewRestClient("TOKEN", s.Client())
	c.BaseURL = s.URL

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		var out1 map[string]interface{}
		errCh <- c.DoJSON(ctx, "GET", "/global", nil, nil, &out1)
	}()

	// Wait until the server has served the global 429; at that point the client should have set a cooldown.
	select {
	case <-served429:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for 429 to be served: %v", ctx.Err())
	}

	// Wait until the client has observed the response and set a global cooldown window.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		c.globalRL.mu.Lock()
		until := c.globalRL.until
		c.globalRL.mu.Unlock()
		if !until.IsZero() && time.Now().Before(until) {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}

	start := time.Now()
	var out2 map[string]interface{}
	if err := c.DoJSON(ctx, "GET", "/other", nil, nil, &out2); err != nil {
		t.Fatalf("DoJSON /other returned error: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 80*time.Millisecond {
		t.Fatalf("expected /other to be delayed by global limiter, elapsed=%s", elapsed)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("DoJSON /global returned error: %v", err)
	}
}

func TestRestClient_DoMultipartWithReason_SendsFieldsAndFile(t *testing.T) {
	t.Parallel()

	fields := map[string]string{
		"payload_json": `{"name":"x"}`,
	}
	fileField := "file"
	fileName := "x.bin"
	fileBytes := []byte("abc123")

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data;") {
			t.Fatalf("expected multipart content-type, got %q", ct)
		}

		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader error: %v", err)
		}

		seenFields := map[string]string{}
		var seenFileName string
		var seenFileBytes []byte

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart error: %v", err)
			}

			b, _ := io.ReadAll(part)
			if part.FileName() != "" {
				seenFileName = part.FileName()
				seenFileBytes = b
			} else {
				seenFields[part.FormName()] = string(b)
			}
			_ = part.Close()
		}

		if seenFields["payload_json"] != fields["payload_json"] {
			t.Fatalf("field payload_json mismatch: got %q want %q", seenFields["payload_json"], fields["payload_json"])
		}
		if seenFileName != fileName {
			t.Fatalf("file name mismatch: got %q want %q", seenFileName, fileName)
		}
		if string(seenFileBytes) != string(fileBytes) {
			t.Fatalf("file bytes mismatch: got %q want %q", string(seenFileBytes), string(fileBytes))
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer s.Close()

	c := NewRestClient("TOKEN", s.Client())
	c.BaseURL = s.URL

	var out map[string]interface{}
	if err := c.DoMultipartWithReason(
		context.Background(),
		"POST",
		"/x",
		nil,
		fields,
		fileField,
		fileName,
		fileBytes,
		&out,
		"because",
	); err != nil {
		t.Fatalf("DoMultipartWithReason returned error: %v", err)
	}
	if v, ok := out["ok"].(bool); !ok || !v {
		t.Fatalf("unexpected response: %#v", out)
	}
}

func TestRestClient_DoMultipartWithReason_EmptyFileFieldStillValidMultipart(t *testing.T) {
	t.Parallel()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("parse content-type: %v", err)
		}
		if params["boundary"] == "" {
			t.Fatalf("expected boundary param in content-type")
		}

		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader error: %v", err)
		}
		// Drain parts; we only care that parsing works.
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart error: %v", err)
			}
			_, _ = io.ReadAll(p)
			_ = p.Close()
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer s.Close()

	c := NewRestClient("TOKEN", s.Client())
	c.BaseURL = s.URL

	var out map[string]interface{}
	if err := c.DoMultipartWithReason(context.Background(), "POST", "/x", nil, map[string]string{"a": "b"}, "", "", nil, &out, ""); err != nil {
		t.Fatalf("DoMultipartWithReason returned error: %v", err)
	}
}
