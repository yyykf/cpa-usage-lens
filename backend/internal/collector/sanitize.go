package collector

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
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

// toEvent 把已解码 CPA 队列条目转成入库明细，供解析单测与兼容逻辑复用。
// collector 主链使用 toEventFromReplay，保证强类型解析发生在 durable save 之后。
// request_id 缺失或 timestamp 解析失败时返回 ok=false，collector 会保留 .rejected 证据。
func toEvent(raw rawQueueItem) (model.UsageEvent, bool) {
	return toEventDecoded(raw, keyFingerprint(raw.APIKey), keyMask(raw.APIKey))
}

func toEventFromReplay(item replayQueueItem) (model.UsageEvent, bool) {
	if item.SanitizationError != "" {
		return model.UsageEvent{}, false
	}
	var raw rawQueueItem
	if err := json.Unmarshal(item.Payload, &raw); err != nil {
		return model.UsageEvent{}, false
	}
	return toEventDecoded(raw, item.KeyFingerprint, item.KeyMask)
}

func toEventDecoded(raw rawQueueItem, keyFingerprintValue, keyMaskValue string) (model.UsageEvent, bool) {
	if raw.RequestID == "" {
		return model.UsageEvent{}, false
	}
	ts, err := time.Parse(time.RFC3339, raw.Timestamp)
	if err != nil {
		return model.UsageEvent{}, false
	}

	ev := model.UsageEvent{
		RequestID:      raw.RequestID,
		EventTS:        ts,
		Source:         raw.Source,
		Provider:       raw.Provider,
		Model:          raw.Model,
		Alias:          raw.Alias,
		Endpoint:       raw.Endpoint,
		AuthType:       raw.AuthType,
		KeyFingerprint: keyFingerprintValue,
		KeyMask:        keyMaskValue,
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
	normalizeCacheAliases(ev.Provider, &ev.Tokens)
	ev.Accounting = legacyAccounting(ev.Tokens, ev.Provider)
	if raw.AccountingVersion == 2 {
		if raw.TokenBreakdown != nil {
			if accounting, ok := canonicalAccounting(*raw.TokenBreakdown); ok {
				ev.Accounting = accounting
			} else {
				ev.Accounting = inconsistentAccounting(max(raw.Tokens.Total, raw.TokenBreakdown.TotalTokens))
			}
		} else {
			ev.Accounting = inconsistentAccounting(raw.Tokens.Total)
		}
		ev.Tokens.Total = ev.Accounting.Tokens.Total
	}
	return ev, true
}

func canonicalAccounting(raw rawTokenBreakdown) (model.Accounting, bool) {
	if raw.SchemaVersion != 2 || (raw.Quality != "complete" && raw.Quality != "unclassified" && raw.Quality != "inconsistent") {
		return model.Accounting{}, false
	}
	values := []int64{raw.TotalTokens, raw.UnclassifiedTokens, raw.Input.TotalTokens, raw.Input.UncachedTokens,
		raw.Input.CacheReadTokens, raw.Input.CacheWriteTokens, raw.Output.TotalTokens,
		raw.Output.NonReasoningTokens, raw.Output.ReasoningTokens}
	for _, value := range values {
		if value < 0 {
			return model.Accounting{}, false
		}
	}
	inputTotal, inputOK := safeTokenSum(raw.Input.UncachedTokens, raw.Input.CacheReadTokens, raw.Input.CacheWriteTokens)
	outputTotal, outputOK := safeTokenSum(raw.Output.NonReasoningTokens, raw.Output.ReasoningTokens)
	total, totalOK := safeTokenSum(raw.Input.TotalTokens, raw.Output.TotalTokens, raw.UnclassifiedTokens)
	if !inputOK || !outputOK || !totalOK || raw.Input.TotalTokens != inputTotal ||
		raw.Output.TotalTokens != outputTotal || raw.TotalTokens != total ||
		(raw.Quality == "complete" && raw.UnclassifiedTokens != 0) {
		return model.Accounting{}, false
	}
	return model.Accounting{Version: 2, Quality: raw.Quality, Tokens: model.CanonicalTokens{
		UncachedInput: raw.Input.UncachedTokens, CacheRead: raw.Input.CacheReadTokens,
		CacheCreation: raw.Input.CacheWriteTokens, NonReasoningOutput: raw.Output.NonReasoningTokens,
		Reasoning: raw.Output.ReasoningTokens, Unclassified: raw.UnclassifiedTokens, Total: raw.TotalTokens,
	}}, true
}

func safeTokenSum(values ...int64) (int64, bool) {
	var total int64
	for _, value := range values {
		if value < 0 || total > int64(^uint64(0)>>1)-value {
			return 0, false
		}
		total += value
	}
	return total, true
}

func inconsistentAccounting(total int64) model.Accounting {
	if total < 0 {
		total = 0
	}
	return model.Accounting{Version: 2, Quality: "inconsistent", Tokens: model.CanonicalTokens{Unclassified: total, Total: total}}
}

func legacyAccounting(tokens model.Tokens, provider string) model.Accounting {
	cacheRead, uncached := tokens.CacheRead, tokens.Input
	switch strings.ToLower(provider) {
	case "codex", "openai":
		cacheRead = max(tokens.Cached, tokens.CacheRead)
		uncached = max(tokens.Input-cacheRead-tokens.CacheCreation, 0)
	case "claude", "anthropic":
		cacheRead = tokens.CacheRead
	default:
		if tokens.Cached > 0 && (tokens.CacheRead == 0 || tokens.CacheRead == tokens.Cached) {
			cacheRead = max(tokens.Cached, tokens.CacheRead)
			uncached = max(tokens.Input-cacheRead-tokens.CacheCreation, 0)
		}
	}
	nonReasoning := tokens.Output
	if strings.EqualFold(provider, "codex") || strings.EqualFold(provider, "openai") {
		nonReasoning = max(tokens.Output-tokens.Reasoning, 0)
	}
	total := uncached + cacheRead + tokens.CacheCreation + nonReasoning + tokens.Reasoning
	return model.Accounting{Version: 1, Quality: "complete", Tokens: model.CanonicalTokens{
		UncachedInput: uncached, CacheRead: cacheRead, CacheCreation: tokens.CacheCreation,
		NonReasoningOutput: nonReasoning, Reasoning: tokens.Reasoning, Total: total,
	}}
}

// normalizeCacheAliases 把 CPA 针对不同 provider 的兼容别名收敛为 Lens 既有口径。
// Codex/OpenAI 用 Cached 表示 input 内含的缓存读，CacheRead 清零避免报表重复展示。
func normalizeCacheAliases(provider string, tokens *model.Tokens) {
	switch strings.ToLower(provider) {
	case "codex", "openai":
		tokens.Cached = max(tokens.Cached, tokens.CacheRead)
		tokens.CacheRead = 0
	case "claude", "anthropic":
		// CPA v7.2.67 同时填 Cached/CacheRead，且 creation-only 时还会用 creation
		// 回填 Cached；Claude 的权威字段是独立的 CacheRead/CacheCreation。
		tokens.Cached = 0
	}
}
