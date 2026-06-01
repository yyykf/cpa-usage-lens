package api

import (
	"testing"
	"time"
)

func TestAuthenticator_CheckPassword(t *testing.T) {
	a, err := NewAuthenticator("s3cret", "signing-key")
	if err != nil {
		t.Fatal(err)
	}
	if !a.CheckPassword("s3cret") {
		t.Error("correct password rejected")
	}
	if a.CheckPassword("wrong") {
		t.Error("wrong password accepted")
	}
}

func TestAuthenticator_TokenRoundTrip(t *testing.T) {
	a, _ := NewAuthenticator("pw", "signing-key")
	tok, err := a.IssueToken(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.ValidateToken(tok); err != nil {
		t.Errorf("valid token rejected: %v", err)
	}
}

func TestAuthenticator_RejectsExpired(t *testing.T) {
	a, _ := NewAuthenticator("pw", "signing-key")
	past := time.Now().Add(-48 * time.Hour) // 24h TTL，48h 前签发 → 过期
	tok, _ := a.IssueToken(past)
	if err := a.ValidateToken(tok); err == nil {
		t.Error("expired token accepted")
	}
}

func TestAuthenticator_RejectsWrongSecret(t *testing.T) {
	a, _ := NewAuthenticator("pw", "key-A")
	b, _ := NewAuthenticator("pw", "key-B")
	tok, _ := a.IssueToken(time.Now())
	if err := b.ValidateToken(tok); err == nil {
		t.Error("token signed with different secret accepted")
	}
}
