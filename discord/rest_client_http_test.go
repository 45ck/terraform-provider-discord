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
