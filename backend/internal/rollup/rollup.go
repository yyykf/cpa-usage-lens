// Package rollup 周期性把 hot 明细聚合进 daily，并清理超期明细（先聚合再删，绝不误删未聚合数据）。
package rollup

import (
	"context"
	"log"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/timeutil"
)

// Store 是 rollup 调度的存储依赖（由 db.DB 实现）。
type Store interface {
	RollupRange(ctx context.Context, startDate, endDate, tz string) error
	DeleteHotBefore(ctx context.Context, beforeDate, tz string) (int64, error)
}

// Scheduler 周期性聚合 + 清理。
type Scheduler struct {
	store         Store
	loc           *time.Location
	retentionDays int
	interval      time.Duration
}

func NewScheduler(store Store, loc *time.Location, retentionDays int, interval time.Duration) *Scheduler {
	return &Scheduler{store: store, loc: loc, retentionDays: retentionDays, interval: interval}
}

// Run 启动调度循环：立即执行一次，再按 interval 周期执行。
func (s *Scheduler) Run(ctx context.Context) {
	s.Tick(ctx, time.Now())
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Tick(ctx, time.Now())
		}
	}
}

// Tick 执行一次聚合 + 清理。
// 安全保证：每次先 rollup 整个保留窗口 [today-retention, today]（幂等，覆盖延迟事件），
// 再删除 < (today-retention) 的明细——删除边界正是 rollup 左端，保证删的天必已聚合。
// 若 rollup 失败则跳过清理，绝不在未聚合时删数据。
func (s *Scheduler) Tick(ctx context.Context, now time.Time) {
	tz := s.loc.String()
	today := timeutil.LocalDate(now, s.loc)
	cutoff := timeutil.DateString(today.AddDate(0, 0, -s.retentionDays), s.loc)
	end := timeutil.DateString(today, s.loc)

	if err := s.store.RollupRange(ctx, cutoff, end, tz); err != nil {
		log.Printf("rollup：聚合 %s~%s 失败（跳过清理）: %v", cutoff, end, err)
		return
	}

	deleted, err := s.store.DeleteHotBefore(ctx, cutoff, tz)
	if err != nil {
		log.Printf("rollup：清理 %s 之前明细失败: %v", cutoff, err)
		return
	}
	if deleted > 0 {
		log.Printf("rollup：已清理 %s 之前的明细 %d 条", cutoff, deleted)
	}
}
