package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

// Buffer 把"已从队列 pop、但尚未确认写入 Supabase"的批次先落盘，
// 确认写库成功后再删除；启动时可恢复残留批次。
// 这是 Supabase 云版独有的防丢保护——pop 不可回放，一次网络抖动就可能丢掉已取出的数据。
type Buffer struct {
	dir string
	seq uint64
}

// NewBuffer 创建缓冲目录。
func NewBuffer(dir string) (*Buffer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Buffer{dir: dir}, nil
}

// Save 把一批 events 落盘，返回批次句柄（文件名）；空批次返回空句柄。
func (b *Buffer) Save(events []model.UsageEvent) (string, error) {
	if len(events) == 0 {
		return "", nil
	}
	data, err := json.Marshal(events)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("batch_%d_%d.json", time.Now().UnixNano(), atomic.AddUint64(&b.seq, 1))
	if err := os.WriteFile(filepath.Join(b.dir, name), data, 0o600); err != nil {
		return "", err
	}
	return name, nil
}

// Commit 删除已确认写库的批次文件。
func (b *Buffer) Commit(handle string) error {
	if handle == "" {
		return nil
	}
	return os.Remove(filepath.Join(b.dir, handle))
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

// Load 读取一个批次文件的 events。
func (b *Buffer) Load(handle string) ([]model.UsageEvent, error) {
	data, err := os.ReadFile(filepath.Join(b.dir, handle))
	if err != nil {
		return nil, err
	}
	var events []model.UsageEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	return events, nil
}
