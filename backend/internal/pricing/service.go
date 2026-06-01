package pricing

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

// Store 是价格服务的存储依赖（由 db.DB 实现）。
type Store interface {
	ListUsedModels(ctx context.Context) ([]string, error)
	UpsertPrices(ctx context.Context, prices []model.ModelPrice) error
	GetPriceMap(ctx context.Context) (map[string]model.ModelPrice, error)
}

// Service 管理价格表：从 LiteLLM 刷新（只刷用过的模型）+ 内存缓存供 query-time 成本计算。
type Service struct {
	store  Store
	client *http.Client
	url    string

	mu    sync.RWMutex
	cache map[string]model.ModelPrice
}

func NewService(store Store, client *http.Client, url string) *Service {
	return &Service{store: store, client: client, url: url, cache: map[string]model.ModelPrice{}}
}

// Refresh 拉取 LiteLLM 价格（只针对用过的模型），upsert 入库并刷新内存缓存。
func (s *Service) Refresh(ctx context.Context) error {
	models, err := s.store.ListUsedModels(ctx)
	if err != nil {
		return err
	}
	if len(models) > 0 {
		prices, err := FetchPrices(ctx, s.client, s.url, models)
		if err != nil {
			return err
		}
		if err := s.store.UpsertPrices(ctx, prices); err != nil {
			return err
		}
	}
	return s.reloadCache(ctx)
}

func (s *Service) reloadCache(ctx context.Context) error {
	m, err := s.store.GetPriceMap(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.cache = m
	s.mu.Unlock()
	return nil
}

// LoadCache 启动时从库加载缓存（不拉 LiteLLM），保证即便离线也有已存价格可用。
func (s *Service) LoadCache(ctx context.Context) error {
	return s.reloadCache(ctx)
}

// Prices 返回当前内存缓存的价格表（query-time 成本用）。
func (s *Service) Prices() map[string]model.ModelPrice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache
}

// RunDaily 启动后立即刷新一次，之后每 24h 刷新。
func (s *Service) RunDaily(ctx context.Context) {
	if err := s.Refresh(ctx); err != nil {
		log.Printf("价格：启动刷新失败（将使用已存价格）: %v", err)
	}
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Refresh(ctx); err != nil {
				log.Printf("价格：定时刷新失败: %v", err)
			}
		}
	}
}
