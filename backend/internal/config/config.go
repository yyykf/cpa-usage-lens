// Package config 从环境变量加载并校验运行配置。
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 是 backend 运行所需的全部配置。
type Config struct {
	CPABaseURL        string
	CPAManagementKey  string
	DatabaseURL       string
	DashboardPassword string
	AuthTokenSecret   string
	PollInterval      time.Duration
	RollupInterval    time.Duration
	BatchSize         int
	HotRetentionDays  int
	Timezone          *time.Location
	BackendPort       string
	CollectorEnabled  bool
}

// Load 读取环境变量，校验必填项并解析类型；任一必填缺失或解析失败即返回错误。
func Load() (*Config, error) {
	cfg := &Config{
		CPABaseURL:        os.Getenv("CPA_BASE_URL"),
		CPAManagementKey:  os.Getenv("CPA_MANAGEMENT_KEY"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		DashboardPassword: os.Getenv("DASHBOARD_PASSWORD"),
		AuthTokenSecret:   os.Getenv("AUTH_TOKEN_SECRET"),
		BackendPort:       getenvDefault("BACKEND_PORT", "8080"),
	}

	required := map[string]string{
		"CPA_BASE_URL":       cfg.CPABaseURL,
		"CPA_MANAGEMENT_KEY": cfg.CPAManagementKey,
		"DATABASE_URL":       cfg.DatabaseURL,
		"DASHBOARD_PASSWORD": cfg.DashboardPassword,
		"AUTH_TOKEN_SECRET":  cfg.AuthTokenSecret,
	}
	for k, v := range required {
		if v == "" {
			return nil, fmt.Errorf("缺少必填环境变量 %s", k)
		}
	}

	pollSec, err := getenvInt("COLLECTOR_POLL_INTERVAL_SECONDS", 3)
	if err != nil {
		return nil, err
	}
	if pollSec < 1 {
		pollSec = 1
	}
	cfg.PollInterval = time.Duration(pollSec) * time.Second

	rollupSec, err := getenvInt("ROLLUP_INTERVAL_SECONDS", 60)
	if err != nil {
		return nil, err
	}
	cfg.RollupInterval = time.Duration(rollupSec) * time.Second

	cfg.BatchSize, err = getenvInt("COLLECTOR_BATCH_SIZE", 200)
	if err != nil {
		return nil, err
	}

	cfg.HotRetentionDays, err = getenvInt("HOT_RETENTION_DAYS", 7)
	if err != nil {
		return nil, err
	}

	tzName := getenvDefault("TIMEZONE", "Asia/Shanghai")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("无效时区 TIMEZONE=%q: %w", tzName, err)
	}
	cfg.Timezone = loc

	// 默认 true：现有 .env 没有该变量也照常采集，零破坏。
	// false 时起一个「只提供查询 API、不采集、不写库」的只读实例（迭代/调试用）。
	cfg.CollectorEnabled = getenvBool("COLLECTOR_ENABLED", true)

	return cfg, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("环境变量 %s 必须是整数，得到 %q", key, v)
	}
	return n, nil
}

// getenvBool 读取布尔型开关：空字符串返回 def；否则用 strconv.ParseBool 容错，
// 解析失败也返回 def（开关类配置不因笔误而阻断启动）。
func getenvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
