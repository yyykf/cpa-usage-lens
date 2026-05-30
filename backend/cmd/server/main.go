// Command server 是 CPA Usage Lens 的后端：后台采集循环 + 定时 rollup/清理 + 价格刷新 + 对外 HTTP API + 鉴权。
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/api"
	"github.com/code4j/cpa-usage-lens/backend/internal/collector"
	"github.com/code4j/cpa-usage-lens/backend/internal/config"
	"github.com/code4j/cpa-usage-lens/backend/internal/db"
	"github.com/code4j/cpa-usage-lens/backend/internal/pricing"
	"github.com/code4j/cpa-usage-lens/backend/internal/rollup"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // 本地开发读 .env；容器内 env 已注入时忽略

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer database.Close()
	log.Printf("已连接 Supabase（时区 %s，保留 %d 天，采集间隔 %s）", cfg.Timezone, cfg.HotRetentionDays, cfg.PollInterval)

	// 价格服务：启动加载缓存 + 后台每日刷新
	priceSvc := pricing.NewService(database, &http.Client{Timeout: 30 * time.Second}, pricing.LiteLLMURL)
	if err := priceSvc.LoadCache(ctx); err != nil {
		log.Printf("价格：加载缓存失败: %v", err)
	}
	go priceSvc.RunDaily(ctx)

	// 采集器：轮询 CPA usage-queue
	buf, err := collector.NewBuffer(getenv("BUFFER_DIR", "./data/buffer"))
	if err != nil {
		log.Fatalf("初始化缓冲目录失败: %v", err)
	}
	cpaClient := collector.NewCPAClient(cfg.CPABaseURL, cfg.CPAManagementKey, &http.Client{Timeout: 20 * time.Second})
	coll := collector.NewCollector(cpaClient, database, buf, cfg.BatchSize, cfg.PollInterval)
	go coll.Run(ctx)

	// rollup 调度：每小时聚合 + 清理
	sched := rollup.NewScheduler(database, cfg.Timezone, cfg.HotRetentionDays, cfg.RollupInterval)
	go sched.Run(ctx)

	// API + 鉴权
	auth, err := api.NewAuthenticator(cfg.DashboardPassword, cfg.AuthTokenSecret)
	if err != nil {
		log.Fatalf("初始化鉴权失败: %v", err)
	}
	apiSrv := api.NewServer(database, priceSvc, priceSvc, auth, cfg.Timezone)
	srv := &http.Server{
		Addr:              ":" + cfg.BackendPort,
		Handler:           apiSrv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("backend 监听 :%s", cfg.BackendPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP 服务异常: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("收到退出信号，正在关闭…")
	cancel() // 停止后台 goroutine
	shutdownCtx, sc := context.WithTimeout(context.Background(), 5*time.Second)
	defer sc()
	_ = srv.Shutdown(shutdownCtx)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
