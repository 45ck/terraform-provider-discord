package discord

import (
	"bytes"
	"encoding/json"
)

func normalizeJSON(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	var v interface{}
	dec := json.NewDecoder(bytes.NewBufferString(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// NormalizeJSON parses a JSON string and re-encodes it into a normalized, stable representation.
// This is useful for storing JSON bodies in Terraform state without formatting noise.
func NormalizeJSON(raw string) (string, error) {
	return normalizeJSON(raw)
}
