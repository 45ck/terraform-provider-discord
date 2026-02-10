package muxserver

import (
	"context"
	"testing"
)

func TestMuxServer_New_DoesNotError(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), "test")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
}
