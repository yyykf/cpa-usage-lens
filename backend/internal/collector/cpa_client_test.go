package collector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPopUsageRaw_PreservesItemWithFieldTypeDrift(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"request_id":"drift","timestamp":"2026-08-09T12:00:00+08:00","api_key":"sk-secret","tokens":{"input_tokens":"unexpected"}}]`))
	}))
	t.Cleanup(srv.Close)

	c := NewCPAClient(srv.URL, "k", srv.Client())
	items, err := c.PopUsageRaw(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("raw items = %d, want 1", len(items))
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(items[0], &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["request_id"]) != `"drift"` || string(fields["api_key"]) != `"sk-secret"` {
		t.Fatalf("raw item changed before buffering: %s", items[0])
	}
}

func TestPopUsageRaw_ParsesAndSendsAuth(t *testing.T) {
	var gotAuth, gotTarget string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotTarget = r.URL.Path + "?" + r.URL.RawQuery
		_, _ = w.Write([]byte(`[{"request_id":"req_1","timestamp":"2026-05-05T12:00:00+08:00","source":"a@x.com","model":"gpt-5.4","tokens":{"total_tokens":30},"api_key":"sk-secret"}]`))
	}))
	defer srv.Close()

	c := NewCPAClient(srv.URL, "mykey", srv.Client())
	items, err := c.PopUsageRaw(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !strings.Contains(string(items[0]), `"request_id":"req_1"`) {
		t.Errorf("parse wrong: %+v", items)
	}
	if gotAuth != "Bearer mykey" {
		t.Errorf("auth header = %q, want 'Bearer mykey'", gotAuth)
	}
	if gotTarget != "/v0/management/usage-queue?count=100" {
		t.Errorf("target = %q", gotTarget)
	}
}

func TestPopUsageRaw_EmptyQueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := NewCPAClient(srv.URL, "k", srv.Client())
	items, err := c.PopUsageRaw(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty, got %d", len(items))
	}
}
