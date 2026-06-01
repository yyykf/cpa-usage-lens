// Package db 管理 Supabase Postgres 连接池与底层健康检查。
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB 包装 pgx 连接池。
type DB struct {
	Pool *pgxpool.Pool
}

// Open 用连接串创建连接池并 Ping 验证连通。
func Open(ctx context.Context, databaseURL string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("解析 DATABASE_URL 失败: %w", err)
	}
	cfg.MaxConns = 8
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("创建连接池失败: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("连接 Supabase 失败: %w", err)
	}
	return &DB{Pool: pool}, nil
}

// Close 关闭连接池。
func (d *DB) Close() {
	if d.Pool != nil {
		d.Pool.Close()
	}
}

// Ping 健康检查。
func (d *DB) Ping(ctx context.Context) error {
	return d.Pool.Ping(ctx)
}
