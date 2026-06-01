package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CPAClient 从 CPA 的 HTTP 管理端点拉取 usage-queue。
type CPAClient struct {
	baseURL string
	key     string
	http    *http.Client
}

// NewCPAClient 构造客户端。baseURL 形如 https://host（不含路径）。
func NewCPAClient(baseURL, key string, client *http.Client) *CPAClient {
	return &CPAClient{baseURL: strings.TrimRight(baseURL, "/"), key: key, http: client}
}

// PopUsage 从队列取最多 count 条（pop 即删，不可回放）；空队列返回长度 0 的切片。
func (c *CPAClient) PopUsage(ctx context.Context, count int) ([]rawQueueItem, error) {
	url := fmt.Sprintf("%s/v0/management/usage-queue?count=%d", c.baseURL, count)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("usage-queue HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var items []rawQueueItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("解析 usage-queue 响应失败: %w", err)
	}
	return items, nil
}
