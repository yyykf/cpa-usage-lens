package collector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPopUsage_ParsesAndSendsAuth(t *testing.T) {
	var gotAuth, gotTarget string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotTarget = r.URL.Path + "?" + r.URL.RawQuery
		_, _ = w.Write([]byte(`[{"request_id":"req_1","timestamp":"2026-05-05T12:00:00+08:00","source":"a@x.com","model":"gpt-5.4","tokens":{"total_tokens":30},"api_key":"sk-secret"}]`))
	}))
	defer srv.Close()

	c := NewCPAClient(srv.URL, "mykey", srv.Client())
	items, err := c.PopUsage(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].RequestID != "req_1" {
		t.Errorf("parse wrong: %+v", items)
	}
	if gotAuth != "Bearer mykey" {
		t.Errorf("auth header = %q, want 'Bearer mykey'", gotAuth)
	}
	if gotTarget != "/v0/management/usage-queue?count=100" {
		t.Errorf("target = %q", gotTarget)
	}
}

func TestPopUsage_EmptyQueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewCPAClient(srv.URL, "k", srv.Client())
	items, err := c.PopUsage(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty, got %d", len(items))
	}
}
