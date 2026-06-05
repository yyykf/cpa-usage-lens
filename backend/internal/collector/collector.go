package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

// Store 是采集器的写入侧依赖（由 db.DB 实现）。
type Store interface {
	InsertEvents(ctx context.Context, events []model.UsageEvent) (int64, error)
	BumpCollectorState(ctx context.Context, s model.CollectorState) error
}

// Collector 轮询 CPA usage-queue，剥敏感、去重写库，并维护采集器状态。
type Collector struct {
	client    *CPAClient
	store     Store
	buffer    *Buffer
	batchSize int
	interval  time.Duration
}

func NewCollector(client *CPAClient, store Store, buffer *Buffer, batchSize int, interval time.Duration) *Collector {
	return &Collector{client: client, store: store, buffer: buffer, batchSize: batchSize, interval: interval}
}

// Run 启动采集循环：先恢复残留缓冲，再按间隔轮询，直到 ctx 取消。
func (c *Collector) Run(ctx context.Context) {
	c.recoverPending(ctx)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.pollOnce(ctx)
		}
	}
}

// recoverPending 重放启动前残留的缓冲批次（上次 pop 了但没确认写库的数据）。
func (c *Collector) recoverPending(ctx context.Context) {
	handles, err := c.buffer.Pending()
	if err != nil {
		log.Printf("采集器：读取缓冲失败: %v", err)
		return
	}
	for _, h := range handles {
		events, err := c.buffer.Load(h)
		if err != nil {
			log.Printf("采集器：缓冲 %s 损坏，隔离为 .corrupt 待人工排查: %v", h, err)
			if qerr := c.buffer.Quarantine(h); qerr != nil {
				log.Printf("采集器：隔离损坏缓冲 %s 失败: %v", h, qerr)
			}
			continue
		}
		if _, err := c.store.InsertEvents(ctx, events); err != nil {
			log.Printf("采集器：恢复缓冲 %s 写库失败（保留待重试）: %v", h, err)
			continue
		}
		_ = c.buffer.Commit(h)
		log.Printf("采集器：已恢复缓冲批次 %s（%d 条）", h, len(events))
	}
}

// pollOnce 执行一次轮询：pop → 剥敏感 → 落盘缓冲 → 写库 → 确认 → 更新状态。
func (c *Collector) pollOnce(ctx context.Context) {
	now := time.Now()
	st := model.CollectorState{LastPollAt: &now}

	items, err := c.client.PopUsage(ctx, c.batchSize)
	if err != nil {
		st.LastError = err.Error()
		st.LastErrorAt = &now
		_ = c.store.BumpCollectorState(ctx, st)
		return
	}
	if len(items) == 0 {
		_ = c.store.BumpCollectorState(ctx, st)
		return
	}

	events := make([]model.UsageEvent, 0, len(items))
	var lastTS time.Time
	var lastID string
	for i := range items {
		ev, ok := toEvent(items[i])
		// toEvent 已就地把明文 api_key 算成指纹+掩码，明文不再有用：
		// 立即清掉队列项里的明文引用，缩短其内存生命周期（即便后续 skip 也清）。
		items[i].APIKey = ""
		if !ok {
			continue
		}
		events = append(events, ev)
		if ev.EventTS.After(lastTS) {
			lastTS = ev.EventTS
			lastID = ev.RequestID // 与 lastTS 保持同一条，诊断信息才一致
		}
	}
	if len(events) == 0 {
		_ = c.store.BumpCollectorState(ctx, st)
		return
	}

	// 先落盘缓冲（防 pop 了但写库失败丢数据），写库确认后再删
	handle, saveErr := c.buffer.Save(events)
	if saveErr != nil {
		log.Printf("采集器：落盘缓冲失败（仍尝试写库，但已失去崩溃保护）: %v", saveErr)
	}

	inserted, err := c.store.InsertEvents(ctx, events)
	if err != nil {
		if saveErr != nil {
			// 缓冲没存上 + 写库失败：这批已 pop 的数据有丢失风险（pop 不可回放），强告警
			st.LastError = fmt.Sprintf("数据丢失风险：缓冲与写库均失败（%d 条）：buffer=%v；insert=%v", len(events), saveErr, err)
		} else {
			st.LastError = err.Error()
		}
		st.LastErrorAt = &now
		_ = c.store.BumpCollectorState(ctx, st)
		return // Save 成功则缓冲保留、下次 recover 重试
	}
	if handle != "" {
		_ = c.buffer.Commit(handle)
	}

	st.EventsIngested = inserted
	if !lastTS.IsZero() {
		st.LastEventTS = &lastTS
	}
	st.LastRequestID = lastID
	_ = c.store.BumpCollectorState(ctx, st)
}
