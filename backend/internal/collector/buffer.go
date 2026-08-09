package collector

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

// Buffer 把"已从队列 pop、但尚未确认写入 Supabase"的批次先落盘，
// 确认写库成功后再删除；启动时可恢复残留批次。
// 这是 Supabase 云版独有的防丢保护——pop 不可回放，一次网络抖动就可能丢掉已取出的数据。
type Buffer struct {
	dir           string
	seq           uint64
	syncDirectory func() error
}

type pendingBatch struct {
	Legacy bool
	Events []model.UsageEvent
	Replay replayBatch
}

// NewBuffer 创建缓冲目录。
func NewBuffer(dir string) (*Buffer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Buffer{dir: dir, syncDirectory: func() error { return syncBufferDirectory(dir) }}, nil
}

// SaveReplayBatch 把已脱敏、尚未规范化的 CPA 批次原子落盘。
func (b *Buffer) SaveReplayBatch(batch replayBatch) (string, error) {
	if len(batch.Items) == 0 {
		return "", nil
	}
	data, err := json.Marshal(batch)
	if err != nil {
		return "", err
	}
	return b.saveJSON(data)
}

func (b *Buffer) saveJSON(data []byte) (string, error) {
	name := fmt.Sprintf("batch_%d_%d.json", time.Now().UnixNano(), atomic.AddUint64(&b.seq, 1))
	final := filepath.Join(b.dir, name)
	tmp := final + ".tmp"

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := b.syncDirectory(); err != nil {
		return name, err
	}
	return name, nil
}

func syncBufferDirectory(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOTSUP) {
		return err
	}
	return nil
}

// Commit 删除已确认写库的批次文件。
func (b *Buffer) Commit(handle string) error {
	if handle == "" {
		return nil
	}
	return os.Remove(filepath.Join(b.dir, handle))
}

// Quarantine 把损坏（无法解析）的批次文件改名为 .corrupt，避免反复加载失败，保留供人工排查。
func (b *Buffer) Quarantine(handle string) error {
	src := filepath.Join(b.dir, handle)
	return os.Rename(src, src+".corrupt")
}

// Reject 保留可解析但无法规范化的已脱敏批次，避免无效队列项静默消失。
func (b *Buffer) Reject(handle string) error {
	if handle == "" {
		return nil
	}
	src := filepath.Join(b.dir, handle)
	return os.Rename(src, src+".rejected")
}

// Pending 列出未提交的批次句柄（启动恢复用）。
func (b *Buffer) Pending() ([]string, error) {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// LoadPending 兼容读取旧版 []UsageEvent 与新版 replayBatch；新写入只使用 replayBatch。
func (b *Buffer) LoadPending(handle string) (pendingBatch, error) {
	data, err := os.ReadFile(filepath.Join(b.dir, handle))
	if err != nil {
		return pendingBatch{}, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return pendingBatch{}, fmt.Errorf("empty buffer file")
	}
	if trimmed[0] == '[' {
		var events []model.UsageEvent
		if err := json.Unmarshal(trimmed, &events); err != nil {
			return pendingBatch{}, err
		}
		return pendingBatch{Legacy: true, Events: events}, nil
	}
	var batch replayBatch
	if err := json.Unmarshal(trimmed, &batch); err != nil {
		return pendingBatch{}, err
	}
	if batch.SchemaVersion != replayBufferSchemaVersion {
		return pendingBatch{}, fmt.Errorf("unsupported replay buffer schema %d", batch.SchemaVersion)
	}
	return pendingBatch{Replay: batch}, nil
}
