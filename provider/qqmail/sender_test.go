package qqmail

import (
	"strings"
	"testing"
)

func TestQQmailSender_Validate_NoAuth(t *testing.T) {
	s := &QQmailSender{}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected error with empty auth")
	}
}

func TestQQmailSender_CheckSendBalance(t *testing.T) {
	s := &QQmailSender{}
	_, err := s.CheckSendBalance()
	if err == nil {
		t.Fatal("expected ErrCapNotSupported")
	}
}

func TestQQmailSender_ListSent(t *testing.T) {
	s := &QQmailSender{}
	_, err := s.ListSent(0, 0)
	if err == nil {
		t.Fatal("expected ErrCapNotSupported")
	}
}

func TestQQmailSender_DeleteSent(t *testing.T) {
	s := &QQmailSender{}
	err := s.DeleteSent(0)
	if err == nil {
		t.Fatal("expected ErrCapNotSupported")
	}
}

func TestQQmailSender_ClearSent(t *testing.T) {
	s := &QQmailSender{}
	err := s.ClearSent()
	if err == nil {
		t.Fatal("expected ErrCapNotSupported")
	}
}

func TestQQmailSender_Validate_WithAuth(t *testing.T) {
	s := &QQmailSender{auth: QQmailAuthData{Username: "test@qq.com", Password: "password"}}
	err := s.Validate()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestBuildSMTPMessage(t *testing.T) {
	msg := buildSMTPMessage("Alice", "alice@qq.com", "bob@example.com", "Bob", "Hello", "Test body", false)
	if !strings.Contains(msg, "From: Alice <alice@qq.com>") {
		t.Fatal("missing From header")
	}
	if !strings.Contains(msg, "To: Bob <bob@example.com>") {
		t.Fatal("missing To header")
	}
	if !strings.Contains(msg, "Subject: Hello") {
		t.Fatal("missing Subject header")
	}
	if !strings.Contains(msg, "text/plain") {
		t.Fatal("expected text/plain Content-Type")
	}
	if !strings.Contains(msg, "\r\nTest body") {
		t.Fatal("missing body content")
	}
}

func TestBuildSMTPMessage_HTML(t *testing.T) {
	msg := buildSMTPMessage("Alice", "alice@qq.com", "bob@example.com", "Bob", "Hello", "<h1>Hi</h1>", true)
	if !strings.Contains(msg, "text/html") {
		t.Fatal("expected text/html Content-Type")
	}
}
