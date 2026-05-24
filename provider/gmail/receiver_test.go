package gmail

import (
	"testing"
)

func TestGmailReceiver_Validate_NoService(t *testing.T) {
	r := &GmailReceiver{}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error with nil service")
	}
}

func TestDecodeBase64(t *testing.T) {
	result := decodeBase64("SGVsbG8gV29ybGQ")
	if result != "Hello World" {
		t.Fatalf("expected 'Hello World', got %q", result)
	}
}
