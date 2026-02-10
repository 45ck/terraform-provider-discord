package discord

import "testing"

func TestNormalizeJSON(t *testing.T) {
	a, err := normalizeJSON("{\"b\":2, \"a\": 1}")
	if err != nil {
		t.Fatalf("normalizeJSON error: %v", err)
	}
	b, err := normalizeJSON("{\"a\":1,\"b\":2}")
	if err != nil {
		t.Fatalf("normalizeJSON error: %v", err)
	}
	if a != b {
		t.Fatalf("expected normalized json to match:\n%s\n%s", a, b)
	}
}
