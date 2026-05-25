package gmail

import (
	"strings"
	"testing"

	"agent-mail/provider"
)

func TestGmailSender_Validate_NoAuth(t *testing.T) {
	s := &GmailSender{}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error with empty sender")
	}
}

func TestGmailSender_CheckSendBalance(t *testing.T) {
	s := &GmailSender{}
	_, err := s.CheckSendBalance()
	if err == nil {
		t.Fatal("expected ErrCapNotSupported")
	}
}

func TestBuildMessage(t *testing.T) {
	msg := provider.BuildSMTPMessage("Alice", "alice@gmail.com", "bob@example.com", "Bob", "Hello", "Test body", false)
	parts := []string{
		"From: Alice <alice@gmail.com>",
		"To: Bob <bob@example.com>",
		"Subject: Hello",
		"Content-Type: text/plain",
		"Test body",
	}
	for _, p := range parts {
		if !strings.Contains(msg, p) {
			t.Fatalf("expected %q in message body", p)
		}
	}
}

func TestBuildMessageHTML(t *testing.T) {
	msg := provider.BuildSMTPMessage("", "", "", "", "", "<p>Hello</p>", true)
	if !strings.Contains(msg, "Content-Type: text/html") {
		t.Fatal("expected HTML Content-Type")
	}
}

func TestBuildMessageHasCRLF(t *testing.T) {
	msg := provider.BuildSMTPMessage("A", "a@b.com", "c@d.com", "C", "S", "body", false)
	if !strings.Contains(msg, "\r\n") {
		t.Fatal("expected CRLF line endings")
	}
}
