package collector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

type recordingStore struct {
	buffer          *Buffer
	events          []model.UsageEvent
	states          []model.CollectorState
	pendingAtInsert int
	insertErr       error
}

func (s *recordingStore) InsertEvents(_ context.Context, events []model.UsageEvent) (int64, error) {
	pending, err := s.buffer.Pending()
	if err != nil {
		return 0, err
	}
	s.pendingAtInsert = len(pending)
	s.events = append([]model.UsageEvent(nil), events...)
	if s.insertErr != nil {
		return 0, s.insertErr
	}
	return int64(len(events)), nil
}

func (s *recordingStore) BumpCollectorState(_ context.Context, state model.CollectorState) error {
	s.states = append(s.states, state)
	return nil
}

func TestCollector_PersistsReplayBeforeNormalizationAndRetainsRejectedItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
          {"request_id":"valid","timestamp":"2026-08-09T12:00:00+08:00","api_key":"sk-valid-secret",
           "tokens":{"input_tokens":2,"output_tokens":1,"total_tokens":3},"accounting_version":2,
           "token_breakdown":{"schema_version":2,"quality":"complete","total_tokens":3,
             "input":{"total_tokens":2,"uncached_tokens":1,"cache_read_tokens":1,"cache_write_tokens":0},
             "output":{"total_tokens":1,"non_reasoning_tokens":0,"reasoning_tokens":1},"unclassified_tokens":0}},
          {"request_id":"invalid-time","timestamp":"not-a-time","api_key":"sk-invalid-secret","tokens":{"total_tokens":5}},
          {"request_id":"type-drift","timestamp":"2026-08-09T12:01:00+08:00","api_key":"sk-drift-secret","tokens":{"input_tokens":"unexpected"}}
        ]`))
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	buffer, err := NewBuffer(dir)
	if err != nil {
		t.Fatal(err)
	}
	store := &recordingStore{buffer: buffer}
	collector := NewCollector(NewCPAClient(srv.URL, "management-secret", srv.Client()), store, buffer, 100, time.Second)
	collector.pollOnce(context.Background())

	if store.pendingAtInsert != 1 {
		t.Fatalf("pending buffers at DB insert = %d, want 1", store.pendingAtInsert)
	}
	if len(store.events) != 1 || store.events[0].RequestID != "valid" {
		t.Fatalf("inserted events = %+v", store.events)
	}
	if store.events[0].Accounting.Version != 2 || store.events[0].Accounting.Quality != "complete" ||
		store.events[0].Accounting.Tokens.CacheRead != 1 || store.events[0].Accounting.Tokens.Reasoning != 1 {
		t.Fatalf("accounting v2 changed across replay: %+v", store.events[0].Accounting)
	}
	if pending, err := buffer.Pending(); err != nil || len(pending) != 0 {
		t.Fatalf("pending after successful DB insert = %v, err=%v", pending, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".rejected") {
		t.Fatalf("buffer entries = %v, want one .rejected artifact", entries)
	}
	data, err := os.ReadFile(dir + "/" + entries[0].Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "invalid-time") || !strings.Contains(string(data), "type-drift") {
		t.Fatalf("rejected artifact lost invalid item: %s", data)
	}
	if strings.Contains(string(data), "sk-valid-secret") || strings.Contains(string(data), "sk-invalid-secret") || strings.Contains(string(data), "sk-drift-secret") {
		t.Fatalf("rejected artifact contains plaintext key: %s", data)
	}
	if len(store.states) == 0 || !strings.Contains(store.states[len(store.states)-1].LastError, "2") {
		t.Fatalf("collector state did not report rejected item: %+v", store.states)
	}
}

func TestCollector_DBFailureKeepsReplayForStartupRecovery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"request_id":"retry","timestamp":"2026-08-09T12:00:00+08:00","api_key":"sk-retry-secret","tokens":{"total_tokens":7}}]`))
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	buffer, err := NewBuffer(dir)
	if err != nil {
		t.Fatal(err)
	}
	store := &recordingStore{buffer: buffer, insertErr: errors.New("db unavailable")}
	collector := NewCollector(NewCPAClient(srv.URL, "management-secret", srv.Client()), store, buffer, 100, time.Second)
	collector.pollOnce(context.Background())

	pending, err := buffer.Pending()
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending after DB failure = %v, err=%v", pending, err)
	}
	data, err := os.ReadFile(dir + "/" + pending[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sk-retry-secret") || !strings.Contains(string(data), "retry") {
		t.Fatalf("pending replay is unsafe or incomplete: %s", data)
	}

	store.insertErr = nil
	collector.recoverPending(context.Background())
	if pending, err := buffer.Pending(); err != nil || len(pending) != 0 {
		t.Fatalf("pending after recovery = %v, err=%v", pending, err)
	}
	if len(store.events) != 1 || store.events[0].RequestID != "retry" || store.events[0].Tokens.Total != 7 {
		t.Fatalf("recovered events = %+v", store.events)
	}
}

func TestCollector_RecoversLegacyNormalizedBuffer(t *testing.T) {
	buffer, err := NewBuffer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	event := model.UsageEvent{RequestID: "legacy", EventTS: time.Date(2026, 8, 9, 4, 0, 0, 0, time.UTC), Tokens: model.Tokens{Total: 9}}
	legacy, err := json.Marshal([]model.UsageEvent{event})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(buffer.dir+"/batch_legacy.json", legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	store := &recordingStore{buffer: buffer}
	collector := NewCollector(nil, store, buffer, 100, time.Second)

	collector.recoverPending(context.Background())

	if len(store.events) != 1 || store.events[0] != event {
		t.Fatalf("legacy recovered events = %+v", store.events)
	}
	if pending, err := buffer.Pending(); err != nil || len(pending) != 0 {
		t.Fatalf("pending legacy buffers = %v, err=%v", pending, err)
	}
}

func TestCollector_QuarantinesUnsupportedReplaySchema(t *testing.T) {
	dir := t.TempDir()
	buffer, err := NewBuffer(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/batch_unsupported.json", []byte(`{"schema_version":99,"items":[{}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &recordingStore{buffer: buffer}
	collector := NewCollector(nil, store, buffer, 100, time.Second)

	collector.recoverPending(context.Background())

	if _, err := os.Stat(dir + "/batch_unsupported.json.corrupt"); err != nil {
		t.Fatalf("unsupported buffer not quarantined: %v", err)
	}
	if len(store.events) != 0 {
		t.Fatalf("unsupported buffer reached DB: %+v", store.events)
	}
	if len(store.states) == 0 || !strings.Contains(store.states[len(store.states)-1].LastError, "unsupported replay buffer schema") {
		t.Fatalf("unsupported buffer not surfaced in collector state: %+v", store.states)
	}
}
