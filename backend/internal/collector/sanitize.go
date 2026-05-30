package collector

import (
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

// toEvent 把 CPA 原始队列条目转成入库用的精简明细，
// 剥离 api_key / response_headers / fail.body 等敏感或大字段（目标结构上根本不含这些字段）。
// request_id 缺失或 timestamp 解析失败时返回 ok=false，调用方应跳过该条。
func toEvent(raw rawQueueItem) (model.UsageEvent, bool) {
	if raw.RequestID == "" {
		return model.UsageEvent{}, false
	}
	ts, err := time.Parse(time.RFC3339, raw.Timestamp)
	if err != nil {
		return model.UsageEvent{}, false
	}

	ev := model.UsageEvent{
		RequestID:       raw.RequestID,
		EventTS:         ts,
		Source:          raw.Source,
		Provider:        raw.Provider,
		Model:           raw.Model,
		Alias:           raw.Alias,
		Endpoint:        raw.Endpoint,
		AuthType:        raw.AuthType,
		Tokens:          model.Tokens(raw.Tokens), // 字段同序同类型，可直接转换
		LatencyMs:       raw.LatencyMs,
		TTFTMs:          raw.TTFTMs,
		Failed:          raw.Failed,
		ReasoningEffort: raw.ReasoningEffort,
		ServiceTier:     raw.ServiceTier,
	}
	ev.AuthIndex = string(raw.AuthIndex)
	if raw.Fail != nil && raw.Fail.StatusCode != nil {
		ev.FailStatusCode = raw.Fail.StatusCode
	}
	return ev, true
}
