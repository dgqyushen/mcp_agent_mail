package qqmail

import (
	"testing"
)

func TestQQmailReceiver_Validate_NoAuth(t *testing.T) {
	r := &QQmailReceiver{}
	err := r.Validate()
	if err == nil {
		t.Fatal("expected error with empty auth")
	}
}
