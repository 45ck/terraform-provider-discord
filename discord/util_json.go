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
