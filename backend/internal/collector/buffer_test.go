package collector

import (
	"testing"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

func TestBuffer_SaveLoadCommitPending(t *testing.T) {
	b, err := NewBuffer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	events := []model.UsageEvent{{RequestID: "r1", Source: "a"}, {RequestID: "r2", Source: "b"}}
	h, err := b.Save(events)
	if err != nil || h == "" {
		t.Fatalf("save: err=%v handle=%q", err, h)
	}

	if pending, _ := b.Pending(); len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}

	loaded, err := b.Load(h)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 || loaded[0].RequestID != "r1" {
		t.Errorf("load mismatch: %+v", loaded)
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
	h, err := b.Save(nil)
	if err != nil || h != "" {
		t.Errorf("empty save should be noop: err=%v handle=%q", err, h)
	}
}
