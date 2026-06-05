package collector

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

// 非 api_key 认证（oauth 等）或空 key 的哨兵值，与 daily_account_usage 列默认保持一致。
const (
	noKeyFingerprint = "none"     // 与 daily 列 default 'none' 对齐，归「其他/非 key 认证」桶
	noKeyMask        = "(no key)" // 界面展示用占位
	keyMaskSuffixLen = 4          // 掩码保留的后缀位数（sk-…后4位）

	// shortKeyMask 是「明文太短、无法安全打码」时的定长占位符：
	// 不含任何原文可辨识片段，避免短 key 把（接近）整段明文回显进掩码（违反「明文绝不入库」）。
	shortKeyMask = "****"
	// noSepMaskPrefix 是无分隔符 key 的固定前缀：不暴露任何原文，仅标识这是一把 key。
	noSepMaskPrefix = "key"
)

// keyFingerprint 计算明文 api_key 的不可逆指纹：sha256 全长小写 hex。
// ⚠️ 采集与回填必须用同一算法，否则同把 key 指纹对不上、被当两把。
// 纯函数：仅用入参算值，绝不把明文写进日志/库/任何结构体。空 key 落哨兵。
func keyFingerprint(apiKey string) string {
	if apiKey == "" {
		return noKeyFingerprint
	}
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])
}

// keyMask 生成可展示的掩码：token 族前缀 + "…" + 后 keyMaskSuffixLen 字符（如 sk-…2216，对齐 PRD「sk-…后4位」）。
// 前缀只露到首个 '-'（含），仅供眼认来源（如 sk-），绝不带 '-' 之后的任何原文字符。
// 无分隔符 key 用固定前缀 "key"（不暴露原文）。
// 安全护栏：当明文短到前缀+后缀之间没有「被真正遮蔽的中间段」时（rune 数 <= 前缀+后缀），
// 返回定长占位符 "****"——不含任何原文可辨识片段，绝不把短 key 原文回显进掩码。
//
// ⚠️ 用 []rune 取后缀，避免非 ASCII key 切在 UTF-8 字节中间（指纹仍用原始 bytes，见 keyFingerprint）。
func keyMask(apiKey string) string {
	if apiKey == "" {
		return noKeyMask
	}

	prefix := keyMaskPrefix(apiKey)
	runes := []rune(apiKey)
	prefixRunes := len([]rune(prefix))

	// 必须存在「前缀 + 后4 之间被遮蔽的中间段」才能安全打码，否则等于回显（接近）整段明文。
	if len(runes) <= prefixRunes+keyMaskSuffixLen {
		return shortKeyMask
	}
	suffix := string(runes[len(runes)-keyMaskSuffixLen:])
	return prefix + "…" + suffix
}

// keyMaskPrefix 取掩码前缀：有 '-' 分隔符时只露「分隔符及之前」（典型 sk-），
// 不暴露 '-' 之后的任何原文字符；无分隔符则用固定前缀 noSepMaskPrefix（不暴露原文）。
func keyMaskPrefix(apiKey string) string {
	for i := 0; i < len(apiKey); i++ {
		if apiKey[i] == '-' {
			return apiKey[:i+1] // 含分隔符本身，如 "sk-"
		}
	}
	return noSepMaskPrefix
}

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
		RequestID: raw.RequestID,
		EventTS:   ts,
		Source:    raw.Source,
		Provider:  raw.Provider,
		Model:     raw.Model,
		Alias:     raw.Alias,
		Endpoint:  raw.Endpoint,
		AuthType:  raw.AuthType,
		// 明文 api_key 仅在此就地算指纹+掩码，算完即弃；明文绝不进 UsageEvent / 库 / 日志。
		KeyFingerprint: keyFingerprint(raw.APIKey),
		KeyMask:        keyMask(raw.APIKey),
		Tokens: model.Tokens{ // 显式逐字段赋值：未来任一 struct 改字段会编译报错，避免静默错位
			Input:         raw.Tokens.Input,
			Output:        raw.Tokens.Output,
			Reasoning:     raw.Tokens.Reasoning,
			Cached:        raw.Tokens.Cached,
			CacheRead:     raw.Tokens.CacheRead,
			CacheCreation: raw.Tokens.CacheCreation,
			Total:         raw.Tokens.Total,
		},
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
