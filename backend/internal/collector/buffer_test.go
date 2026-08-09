package collector

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

func TestNewReplayBatchFromRaw_RedactsBeforeTypedDecode(t *testing.T) {
	raw := json.RawMessage(`{
      "request_id":"drift","timestamp":"2026-08-09T12:00:00+08:00",
      "api_key":"sk-raw-secret","response_headers":{"authorization":"header-secret"},
      "fail":{"status_code":500,"body":"failure-secret"},
      "tokens":{"input_tokens":"unexpected"},"future_field":{"kept":true}
    }`)

	rawItems := []json.RawMessage{raw}
	batch := newReplayBatchFromRaw(rawItems, time.Now())
	if rawItems[0] != nil {
		t.Fatal("plaintext raw item must be cleared after sanitized envelope construction")
	}
	if len(batch.Items) != 1 {
		t.Fatalf("batch items = %d, want 1", len(batch.Items))
	}
	item := batch.Items[0]
	serialized, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"sk-raw-secret", "header-secret", "failure-secret", `"api_key"`, `"response_headers"`, `"body"`} {
		if strings.Contains(string(serialized), forbidden) {
			t.Fatalf("sanitized raw item contains %q: %s", forbidden, serialized)
		}
	}
	if !strings.Contains(string(serialized), `"future_field"`) || !strings.Contains(string(serialized), `"unexpected"`) {
		t.Fatalf("raw payload fields lost before durable save: %s", serialized)
	}
	if item.KeyFingerprint != keyFingerprint("sk-raw-secret") || item.KeyMask != keyMask("sk-raw-secret") {
		t.Fatalf("key identity = %q/%q", item.KeyFingerprint, item.KeyMask)
	}
}

func TestNewReplayBatchFromRaw_RejectsInvalidAPIKeyTypeWithoutPersistingIt(t *testing.T) {
	rawItems := []json.RawMessage{json.RawMessage(`{"request_id":"bad-key","timestamp":"2026-08-09T12:00:00+08:00","api_key":{"secret":"must-not-persist"}}`)}
	batch := newReplayBatchFromRaw(rawItems, time.Now())
	item := batch.Items[0]
	serialized, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), "must-not-persist") || strings.Contains(string(serialized), `"api_key"`) {
		t.Fatalf("invalid api_key leaked: %s", serialized)
	}
	if item.SanitizationError == "" {
		t.Fatal("invalid api_key type must remain observable and reject normalization")
	}
	if _, ok := toEventFromReplay(item); ok {
		t.Fatal("item with invalid api_key type must not become a usage event")
	}
}

func TestBuffer_SaveReplayBatchRedactsSecretsAndRetainsAccounting(t *testing.T) {
	dir := t.TempDir()
	b, err := NewBuffer(dir)
	if err != nil {
		t.Fatal(err)
	}

	items := []rawQueueItem{{
		Timestamp: "2026-08-09T12:00:00+08:00", RequestID: "r-v2", Provider: "codex",
		APIKey: "sk-plaintext-secret", ResponseHeaders: []byte(`{"authorization":"secret-header"}`),
		Fail:   &rawFail{Body: "secret-failure-body"},
		Tokens: rawTokens{Input: 100, Output: 20, Total: 120}, AccountingVersion: 2,
		TokenBreakdown: &rawTokenBreakdown{SchemaVersion: 2, Quality: "complete", TotalTokens: 120,
			Input:  rawInputBreakdown{TotalTokens: 100, UncachedTokens: 60, CacheReadTokens: 30, CacheWriteTokens: 10},
			Output: rawOutputBreakdown{TotalTokens: 20, NonReasoningTokens: 5, ReasoningTokens: 15}},
	}}
	raw, err := json.Marshal(items[0])
	if err != nil {
		t.Fatal(err)
	}
	batch := newReplayBatchFromRaw([]json.RawMessage{raw}, time.Date(2026, 8, 9, 4, 0, 0, 0, time.UTC))
	handle, err := b.SaveReplayBatch(batch)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dir + "/" + handle)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"sk-plaintext-secret", "secret-header", "secret-failure-body"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("buffer contains secret %q: %s", secret, data)
		}
	}
	for _, forbiddenField := range []string{`"api_key"`, `"response_headers"`, `"body"`} {
		if strings.Contains(string(data), forbiddenField) {
			t.Fatalf("buffer contains forbidden field %s: %s", forbiddenField, data)
		}
	}

	pending, err := b.LoadPending(handle)
	if err != nil {
		t.Fatal(err)
	}
	loaded := pending.Replay
	if loaded.SchemaVersion != replayBufferSchemaVersion || len(loaded.Items) != 1 {
		t.Fatalf("loaded batch = %+v", loaded)
	}
	item := loaded.Items[0]
	if item.KeyFingerprint != keyFingerprint("sk-plaintext-secret") || item.KeyMask != keyMask("sk-plaintext-secret") {
		t.Fatalf("sanitized key identity = %q/%q", item.KeyFingerprint, item.KeyMask)
	}
	event, ok := toEventFromReplay(item)
	if !ok || event.Accounting.Version != 2 || event.Accounting.Tokens.Reasoning != 15 {
		t.Fatalf("accounting v2 lost: ok=%v event=%+v", ok, event)
	}
}

func TestReplayQueueItem_ToEventPreservesSanitizedIdentityAndAccounting(t *testing.T) {
	raw := rawQueueItem{
		Timestamp: "2026-08-09T12:00:00+08:00", RequestID: "r-v2", Provider: "codex",
		APIKey: "sk-replay-secret", Tokens: rawTokens{Input: 100, Output: 20, Total: 120},
		AccountingVersion: 2,
		TokenBreakdown: &rawTokenBreakdown{SchemaVersion: 2, Quality: "complete", TotalTokens: 120,
			Input:  rawInputBreakdown{TotalTokens: 100, UncachedTokens: 60, CacheReadTokens: 30, CacheWriteTokens: 10},
			Output: rawOutputBreakdown{TotalTokens: 20, NonReasoningTokens: 5, ReasoningTokens: 15}},
	}
	want, ok := toEvent(raw)
	if !ok {
		t.Fatal("direct normalization failed")
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	batch := newReplayBatchFromRaw([]json.RawMessage{encoded}, time.Now())
	got, ok := toEventFromReplay(batch.Items[0])
	if !ok {
		t.Fatal("replay normalization failed")
	}
	if got != want {
		t.Fatalf("replayed event = %+v, want %+v", got, want)
	}
}

func TestBuffer_LoadReplayBatchRejectsUnsupportedSchema(t *testing.T) {
	dir := t.TempDir()
	b, err := NewBuffer(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/batch_unsupported.json", []byte(`{"schema_version":99,"items":[{}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := b.LoadPending("batch_unsupported.json"); err == nil {
		t.Fatal("unsupported replay schema must fail")
	}
}

func TestBuffer_SaveReplayBatchSyncsDirectoryAfterRename(t *testing.T) {
	dir := t.TempDir()
	b, err := NewBuffer(dir)
	if err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("directory sync failed")
	b.syncDirectory = func() error { return wantErr }
	batch := newReplayBatchFromRaw([]json.RawMessage{json.RawMessage(`{"request_id":"r1"}`)}, time.Now())

	handle, err := b.SaveReplayBatch(batch)
	if !errors.Is(err, wantErr) {
		t.Fatalf("save error = %v, want %v", err, wantErr)
	}
	if handle == "" {
		t.Fatal("renamed durable file handle must survive directory sync error")
	}
	if _, err := os.Stat(dir + "/" + handle); err != nil {
		t.Fatalf("renamed file missing after directory sync error: %v", err)
	}
}

func TestBuffer_LoadPendingReadsLegacyEventBatch(t *testing.T) {
	b, err := NewBuffer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := json.Marshal([]model.UsageEvent{{RequestID: "legacy-r1", Source: "legacy"}})
	if err != nil {
		t.Fatal(err)
	}
	handle := "batch_legacy.json"
	if err := os.WriteFile(b.dir+"/"+handle, legacy, 0o600); err != nil {
		t.Fatal(err)
	}

	batch, err := b.LoadPending(handle)
	if err != nil {
		t.Fatal(err)
	}
	if !batch.Legacy || len(batch.Events) != 1 || batch.Events[0].RequestID != "legacy-r1" {
		t.Fatalf("legacy pending batch = %+v", batch)
	}
}

func TestBuffer_SaveCommitPending(t *testing.T) {
	b, err := NewBuffer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	batch := newReplayBatchFromRaw([]json.RawMessage{json.RawMessage(`{"request_id":"r1"}`)}, time.Now())
	h, err := b.SaveReplayBatch(batch)
	if err != nil || h == "" {
		t.Fatalf("save: err=%v handle=%q", err, h)
	}

	if pending, _ := b.Pending(); len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}

	if err := b.Commit(h); err != nil {
		t.Fatal(err)
	}
	if pending, _ := b.Pending(); len(pending) != 0 {
		t.Errorf("expected 0 pending after commit, got %d", len(pending))
	}
}

func TestBuffer_SaveEmptyIsNoop(t *testing.T) {
	b, _ := NewBuffer(t.TempDir())
	h, err := b.SaveReplayBatch(replayBatch{SchemaVersion: replayBufferSchemaVersion})
	if err != nil || h != "" {
		t.Errorf("empty save should be noop: err=%v handle=%q", err, h)
	}
}
