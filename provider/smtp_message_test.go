package provider

import (
	"strings"
	"testing"
)

func TestBuildSMTPMessagePlain(t *testing.T) {
	msg := BuildSMTPMessage("Me", "me@example.com", "you@example.com", "You", "Test", "Hello world", false)
	if !strings.Contains(msg, "From: Me <me@example.com>") {
		t.Error("missing From header")
	}
	if !strings.Contains(msg, "To: You <you@example.com>") {
		t.Error("missing To header")
	}
	if !strings.Contains(msg, "Subject: Test") {
		t.Error("missing Subject header")
	}
	if !strings.Contains(msg, "text/plain") {
		t.Error("expected text/plain content type")
	}
	if !strings.Contains(msg, "\r\n\r\nHello world") {
		t.Error("missing message body")
	}
}

func TestBuildSMTPMessageHTML(t *testing.T) {
	msg := BuildSMTPMessage("Me", "me@example.com", "you@example.com", "You", "Test", "<h1>Hello</h1>", true)
	if !strings.Contains(msg, "text/html") {
		t.Error("expected text/html content type")
	}
}
