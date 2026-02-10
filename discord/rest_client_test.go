package discord

import "testing"

func TestIsDiscordHTTPStatus(t *testing.T) {
	err := &DiscordHTTPError{StatusCode: 404}
	if !IsDiscordHTTPStatus(err, 404) {
		t.Fatalf("expected true for 404")
	}
	if IsDiscordHTTPStatus(err, 400) {
		t.Fatalf("expected false for 400")
	}
	if IsDiscordHTTPStatus(nil, 404) {
		t.Fatalf("expected false for nil")
	}
}
