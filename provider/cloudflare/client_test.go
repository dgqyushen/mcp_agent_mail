package cloudflare

import (
	"testing"
)

func TestClient_Validate_NoConnection(t *testing.T) {
	c := New("http://127.0.0.1:19999", "fake-jwt", "")
	err := c.Validate()
	if err == nil {
		t.Error("expected connection error for unreachable URL")
	}
}

func TestClient_New(t *testing.T) {
	c := New("https://example.com", "jwt-token", "pass123")
	if c.baseURL != "https://example.com" {
		t.Errorf("expected baseURL https://example.com, got %s", c.baseURL)
	}
	if c.jwt != "jwt-token" {
		t.Errorf("expected jwt jwt-token, got %s", c.jwt)
	}
	if c.sitePass != "pass123" {
		t.Errorf("expected sitePass pass123, got %s", c.sitePass)
	}
	if c.httpClient == nil {
		t.Error("expected httpClient to be initialized")
	}
}

func TestClient_CheckSendBalance(t *testing.T) {
	c := New("", "", "")
	_, err := c.CheckSendBalance()
	if err == nil {
		t.Error("expected error for empty base URL")
	}
}

func TestClient_DeleteSent_NotSupported(t *testing.T) {
	c := New("", "", "")
	err := c.DeleteSent(1)
	if err == nil {
		t.Error("expected error for unsupported operation")
	}
}

func TestClient_ClearSent_NotSupported(t *testing.T) {
	c := New("", "", "")
	err := c.ClearSent()
	if err == nil {
		t.Error("expected error for unsupported operation")
	}
}

func TestClient_ListSent_NotSupported(t *testing.T) {
	c := New("", "", "")
	_, err := c.ListSent(10, 0)
	if err == nil {
		t.Error("expected error for unsupported operation")
	}
}
