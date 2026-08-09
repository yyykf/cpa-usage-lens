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

// Collector 轮询 CPA usage-queue，剥敏感、按复合键（request_id+event_ts+total_tokens）去重写库，并维护采集器状态。
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
		c.recordRecoveryError(ctx, fmt.Sprintf("读取缓冲失败: %v", err))
		return
	}
	for _, h := range handles {
		batch, err := c.buffer.LoadPending(h)
		if err != nil {
			log.Printf("采集器：缓冲 %s 损坏，隔离为 .corrupt 待人工排查: %v", h, err)
			message := fmt.Sprintf("缓冲 %s 无法恢复: %v", h, err)
			if qerr := c.buffer.Quarantine(h); qerr != nil {
				log.Printf("采集器：隔离损坏缓冲 %s 失败: %v", h, qerr)
				message += fmt.Sprintf("；隔离失败: %v", qerr)
			}
			c.recordRecoveryError(ctx, message)
			continue
		}
		result, err := c.processPending(ctx, h, batch)
		if err != nil {
			log.Printf("采集器：恢复缓冲 %s 写库失败（保留待重试）: %v", h, err)
			c.recordRecoveryError(ctx, fmt.Sprintf("恢复缓冲 %s 失败: %v", h, err))
			continue
		}
		if result.rejected > 0 {
			c.recordRecoveryError(ctx, fmt.Sprintf("恢复缓冲 %s 时有 %d 条记录无法规范化，已保留 .rejected", h, result.rejected))
		}
		log.Printf("采集器：已恢复缓冲批次 %s（入库 %d 条，拒绝 %d 条）", h, result.inserted, result.rejected)
	}
}

func (c *Collector) recordRecoveryError(ctx context.Context, message string) {
	now := time.Now()
	_ = c.store.BumpCollectorState(ctx, model.CollectorState{LastError: message, LastErrorAt: &now})
}

// pollOnce 执行一次轮询：pop → 剥敏感 → 落盘缓冲 → 写库 → 确认 → 更新状态。
func (c *Collector) pollOnce(ctx context.Context) {
	now := time.Now()
	st := model.CollectorState{LastPollAt: &now}

	items, err := c.client.PopUsageRaw(ctx, c.batchSize)
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

	// pop 后只做不可避免的密钥脱敏；原始协议字段先落盘，再做解析和 accounting 校验。
	batch := newReplayBatchFromRaw(items, now)
	handle, saveErr := c.buffer.SaveReplayBatch(batch)
	if saveErr != nil {
		log.Printf("采集器：落盘缓冲失败（仍尝试写库，但已失去崩溃保护）: %v", saveErr)
	}

	result, err := c.processPending(ctx, handle, pendingBatch{Replay: batch})
	if err != nil {
		if saveErr != nil {
			// 缓冲没存上 + 写库失败：这批已 pop 的数据有丢失风险（pop 不可回放），强告警
			st.LastError = fmt.Sprintf("数据丢失风险：缓冲与处理均失败（%d 条）：buffer=%v；process=%v", len(items), saveErr, err)
		} else {
			st.LastError = err.Error()
		}
		st.LastErrorAt = &now
		_ = c.store.BumpCollectorState(ctx, st)
		return
	}
	if saveErr != nil {
		st.LastError = fmt.Sprintf("数据丢失风险：落盘缓冲失败（%d 条）：%v", len(items), saveErr)
		st.LastErrorAt = &now
	} else if result.rejected > 0 {
		st.LastError = fmt.Sprintf("%d 条队列记录无法规范化，已保留 .rejected 待排查", result.rejected)
		st.LastErrorAt = &now
	}

	st.EventsIngested = result.inserted
	if !result.lastTS.IsZero() {
		st.LastEventTS = &result.lastTS
	}
	st.LastRequestID = result.lastID
	_ = c.store.BumpCollectorState(ctx, st)
}

type pendingResult struct {
	inserted int64
	rejected int
	lastTS   time.Time
	lastID   string
}

func (c *Collector) processPending(ctx context.Context, handle string, batch pendingBatch) (pendingResult, error) {
	result := pendingResult{}
	events := batch.Events
	if !batch.Legacy {
		events = make([]model.UsageEvent, 0, len(batch.Replay.Items))
		for _, item := range batch.Replay.Items {
			ev, ok := toEventFromReplay(item)
			if !ok {
				result.rejected++
				continue
			}
			events = append(events, ev)
		}
	}
	for _, ev := range events {
		if ev.EventTS.After(result.lastTS) {
			result.lastTS = ev.EventTS
			result.lastID = ev.RequestID
		}
	}
	if len(events) > 0 {
		inserted, err := c.store.InsertEvents(ctx, events)
		if err != nil {
			return pendingResult{}, err
		}
		result.inserted = inserted
	}
	if handle == "" {
		return result, nil
	}
	if result.rejected > 0 {
		if err := c.buffer.Reject(handle); err != nil {
			return pendingResult{}, fmt.Errorf("保留拒绝批次 %s: %w", handle, err)
		}
		return result, nil
	}
	if err := c.buffer.Commit(handle); err != nil {
		return pendingResult{}, fmt.Errorf("提交缓冲 %s: %w", handle, err)
	}
	return result, nil
}
