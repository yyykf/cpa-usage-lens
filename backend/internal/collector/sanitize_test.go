package collector

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

func int32Ptr(i int32) *int32 { return &i }

func TestToEvent_StripsSensitiveAndParses(t *testing.T) {
	raw := rawQueueItem{
		Timestamp: "2026-05-05T12:00:00+08:00",
		Source:    "user@example.com",
		Model:     "gpt-5.4",
		RequestID: "req_123",
		APIKey:    "sk-secret-should-be-dropped",
		Tokens:    rawTokens{Input: 10, Output: 20, CacheRead: 3, Total: 33},
		Fail:      &rawFail{StatusCode: int32Ptr(429), Body: "should-be-dropped"},
	}
	ev, ok := toEvent(raw)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if ev.RequestID != "req_123" {
		t.Errorf("request_id = %q", ev.RequestID)
	}
	if ev.Tokens.Input != 10 || ev.Tokens.Total != 33 || ev.Tokens.CacheRead != 3 {
		t.Errorf("tokens mismatch: %+v", ev.Tokens)
	}
	if ev.FailStatusCode == nil || *ev.FailStatusCode != 429 {
		t.Errorf("fail status not mapped")
	}
	if ev.EventTS.IsZero() {
		t.Error("timestamp not parsed")
	}
	// 注：model.UsageEvent 结构上不含 api_key / response_headers / fail.body，编译期即保证不入库。
}

// 守门铁律：明文 api_key 经 toEvent 后必须 ① 不出现在 UsageEvent 任何字段，
// ② 只以 sha256 指纹 + 掩码两种不可逆形态存在。指纹算法须与回填脚本一致。
func TestToEvent_APIKeyFingerprintAndMaskNeverLeakPlaintext(t *testing.T) {
	const plaintext = "sk-secret-should-be-dropped"
	raw := rawQueueItem{
		Timestamp: "2026-05-05T12:00:00+08:00",
		Source:    "user@example.com",
		Model:     "gpt-5.4",
		RequestID: "req_123",
		APIKey:    plaintext,
		Tokens:    rawTokens{Input: 10, Total: 10},
	}
	ev, ok := toEvent(raw)
	if !ok {
		t.Fatal("expected ok=true")
	}

	// ① 指纹 = sha256(明文) 全长小写 hex（与 supabase 回填脚本必须同算法，否则同 key 对不上）
	sum := sha256.Sum256([]byte(plaintext))
	want := hex.EncodeToString(sum[:])
	if ev.KeyFingerprint != want {
		t.Errorf("fingerprint = %q, want %q", ev.KeyFingerprint, want)
	}
	if len(ev.KeyFingerprint) != 64 {
		t.Errorf("fingerprint should be 64 hex chars, got %d", len(ev.KeyFingerprint))
	}

	// ② 掩码格式：token 族前缀…后4位（如 sk-…pped），且不得是整段明文
	if !strings.Contains(ev.KeyMask, "…") || !strings.HasSuffix(ev.KeyMask, "pped") {
		t.Errorf("mask format wrong: %q", ev.KeyMask)
	}

	// ③ 最硬的一条：UsageEvent 任何字段都不得含明文 api_key 子串（指纹/掩码均不可逆，不含整段明文）
	dump := fmt.Sprintf("%+v", ev)
	if strings.Contains(dump, plaintext) {
		t.Fatalf("plaintext api_key leaked into UsageEvent: %s", dump)
	}
}

// 非 api_key 认证（oauth 等）/ 空 key → 落哨兵，与 daily 列默认一致，归「其他」桶不丢数据。
func TestToEvent_NoAPIKeyFallsBackToSentinel(t *testing.T) {
	raw := rawQueueItem{
		Timestamp: "2026-05-05T12:00:00+08:00",
		Source:    "user@example.com",
		Model:     "gpt-5.4",
		RequestID: "req_oauth",
		AuthType:  "oauth",
		// APIKey 缺省为空
	}
	ev, ok := toEvent(raw)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if ev.KeyFingerprint != noKeyFingerprint {
		t.Errorf("empty key fingerprint = %q, want %q", ev.KeyFingerprint, noKeyFingerprint)
	}
	if ev.KeyMask != noKeyMask {
		t.Errorf("empty key mask = %q, want %q", ev.KeyMask, noKeyMask)
	}
}

func TestKeyFingerprint(t *testing.T) {
	// 空 key → 哨兵；同一 key 两次结果稳定（确定性，回填可复算）
	if got := keyFingerprint(""); got != noKeyFingerprint {
		t.Errorf("empty -> %q, want %q", got, noKeyFingerprint)
	}
	a, b := keyFingerprint("sk-abc"), keyFingerprint("sk-abc")
	if a != b {
		t.Errorf("fingerprint not deterministic: %q vs %q", a, b)
	}
	if keyFingerprint("sk-abc") == keyFingerprint("sk-xyz") {
		t.Error("different keys should yield different fingerprints")
	}
}

func TestKeyMask(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"empty", "", noKeyMask},
		{"typical sk", "sk-e305abcdef2216", "sk-…2216"},          // 前缀只到 sk-，… 后4位，对齐 PRD「sk-…后4位」
		{"short no separator", "abc", shortKeyMask},              // rune 数 <= 前缀+后4 → 定长占位，绝不回显原文
		{"short with sk", "sk-1", shortKeyMask},                  // sk- + 后4 已覆盖整串、无遮蔽中段 → 占位 ****
		{"sk just over threshold", "sk-12345", "sk-…2345"},       // 8 runes > 3+4，'1' 被真正遮蔽 → 可安全打码
		{"no separator long", "abcdefghij", "key…ghij"},          // 无分隔符用固定前缀 key（不暴露原文）+ 后4
		{"unicode suffix uses runes", "sk-长度测试键值😀好", "sk-…键值😀好"}, // 后4字符按 rune 取，不切坏 UTF-8
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := keyMask(c.in)
			if got != c.want {
				t.Errorf("keyMask(%q) = %q, want %q", c.in, got, c.want)
			}
			// 任何掩码都不得等于整段明文（空 key 例外，走哨兵）
			if c.in != "" && got == c.in {
				t.Errorf("mask must not equal plaintext: %q", got)
			}
		})
	}
}

// 守门：短 key 掩码必须是不含任何原文片段的定长占位符，绝不回显（接近）整段明文。
func TestKeyMask_ShortKeyNeverLeaksPlaintext(t *testing.T) {
	for _, in := range []string{"a", "abc", "sk-", "sk-1", "sk-12", "x-y", "1234567"} {
		got := keyMask(in)
		if got != shortKeyMask {
			t.Errorf("keyMask(%q) = %q, want占位符 %q", in, got, shortKeyMask)
		}
		// 占位符里不得出现明文里「独有」的尾部字符（前缀 sk-/key 不算原文敏感片段，这里校验整串不被回显）
		if strings.Contains(got, in) {
			t.Errorf("short-key mask %q still contains plaintext %q", got, in)
		}
	}
}

// 编译期 + 运行期双保险：UsageEvent 字段名里不含 "APIKey"/"api_key"，结构上无处存明文。
func TestUsageEventHasNoPlaintextKeyField(t *testing.T) {
	dump := fmt.Sprintf("%+v", model.UsageEvent{KeyFingerprint: "fp", KeyMask: "mask"})
	for _, banned := range []string{"APIKey", "api_key"} {
		if strings.Contains(dump, banned) {
			t.Errorf("UsageEvent exposes a plaintext-key field %q: %s", banned, dump)
		}
	}
}

// 本次修复的语义守门：CPA ws 路径同一连接的多轮共享同一 request_id，靠每轮独立的
// timestamp（→ event_ts）区分。toEvent 必须为各轮构造出去重复合键
// (request_id, event_ts, total_tokens) 各字段正确、且轮与轮之间复合键互不相同的 event，
// 这样入库侧 ON CONFLICT (request_id, event_ts, total_tokens) DO NOTHING 才不会把后续轮误吞。
func TestToEvent_SameRequestIDDistinctComposite(t *testing.T) {
	// 同一 ws 连接的两轮：request_id 相同，timestamp（每轮各自 time.Now()）与 total 不同。
	rounds := []rawQueueItem{
		{
			Timestamp: "2026-05-05T12:00:00.111111111+08:00",
			Source:    "user@example.com",
			Model:     "gpt-5.4",
			RequestID: "ws_conn_1",
			Tokens:    rawTokens{Input: 10, Output: 5, Total: 15},
		},
		{
			Timestamp: "2026-05-05T12:00:03.222222222+08:00",
			Source:    "user@example.com",
			Model:     "gpt-5.4",
			RequestID: "ws_conn_1",
			Tokens:    rawTokens{Input: 100, Output: 40, CacheRead: 90, Total: 140},
		},
	}

	type compositeKey struct {
		requestID string
		eventTS   time.Time
		total     int64
	}
	seen := make(map[compositeKey]bool, len(rounds))
	for i, raw := range rounds {
		ev, ok := toEvent(raw)
		if !ok {
			t.Fatalf("round %d: expected ok=true", i)
		}
		if ev.RequestID != "ws_conn_1" {
			t.Errorf("round %d: request_id = %q, want shared ws_conn_1", i, ev.RequestID)
		}
		// 亚秒必须被保留（区分多轮的关键）：time.Parse(RFC3339) 自动吸收纳秒。
		if ev.EventTS.Nanosecond() == 0 {
			t.Errorf("round %d: sub-second lost, event_ts=%v", i, ev.EventTS)
		}
		k := compositeKey{ev.RequestID, ev.EventTS, ev.Tokens.Total}
		if seen[k] {
			t.Fatalf("round %d: composite key collides with an earlier round: %+v", i, k)
		}
		seen[k] = true
	}
	// 两轮各成一把复合键 → 入库不会被 DO NOTHING 去重吞掉（修漏记）。
	if len(seen) != len(rounds) {
		t.Fatalf("expected %d distinct composite keys, got %d", len(rounds), len(seen))
	}
}

// 崩溃恢复幂等的语义守门：buffer 重放的是同一条物理记录，toEvent 对完全相同的原始项
// 必产出完全相同的复合键 → 入库侧 DO NOTHING 跳过、不产生重复行。
func TestToEvent_IdenticalRawYieldsIdenticalComposite(t *testing.T) {
	raw := rawQueueItem{
		Timestamp: "2026-05-05T12:00:00.123456789+08:00",
		Source:    "user@example.com",
		Model:     "gpt-5.4",
		RequestID: "ws_conn_1",
		Tokens:    rawTokens{Input: 10, Output: 5, Total: 15},
	}
	a, okA := toEvent(raw)
	b, okB := toEvent(raw)
	if !okA || !okB {
		t.Fatalf("expected ok=true (a=%v b=%v)", okA, okB)
	}
	if a.RequestID != b.RequestID || !a.EventTS.Equal(b.EventTS) || a.Tokens.Total != b.Tokens.Total {
		t.Errorf("replay must yield identical composite key: a=(%q,%v,%d) b=(%q,%v,%d)",
			a.RequestID, a.EventTS, a.Tokens.Total, b.RequestID, b.EventTS, b.Tokens.Total)
	}
}

func TestToEvent_MissingRequestID(t *testing.T) {
	if _, ok := toEvent(rawQueueItem{Timestamp: "2026-05-05T12:00:00+08:00"}); ok {
		t.Error("expected skip when request_id missing")
	}
}

func TestToEvent_BadTimestamp(t *testing.T) {
	if _, ok := toEvent(rawQueueItem{RequestID: "x", Timestamp: "not-a-time"}); ok {
		t.Error("expected skip on bad timestamp")
	}
}

func TestToEvent_CodexCPA7267CacheReadAliasesNormalizeOnce(t *testing.T) {
	raw := rawQueueItem{
		Timestamp: "2026-07-12T12:00:00+08:00",
		Provider:  "codex",
		Model:     "gpt-5.6-sol",
		RequestID: "req-cache-alias",
		Tokens: rawTokens{
			Input: 100, Output: 20, Cached: 30, CacheRead: 30, CacheCreation: 40, Total: 120,
		},
	}
	ev, ok := toEvent(raw)
	if !ok {
		t.Fatal("expected event")
	}
	if ev.Tokens.Cached != 30 || ev.Tokens.CacheRead != 0 {
		t.Errorf("canonical cache read = (cached:%d read:%d), want (30,0)", ev.Tokens.Cached, ev.Tokens.CacheRead)
	}
	if ev.Tokens.CacheCreation != 40 || ev.Tokens.Input != 100 {
		t.Errorf("unrelated token fields changed: %+v", ev.Tokens)
	}
}

func TestToEvent_CodexOldCPAMissingNewAliasesRemainsCompatible(t *testing.T) {
	raw := rawQueueItem{
		Timestamp: "2026-07-12T12:00:00+08:00",
		Provider:  "codex",
		Model:     "gpt-5.5",
		RequestID: "req-old-cpa",
		Tokens:    rawTokens{Input: 100, Output: 20, Cached: 30, Total: 120},
	}
	ev, ok := toEvent(raw)
	if !ok {
		t.Fatal("expected event")
	}
	if ev.Tokens.Cached != 30 || ev.Tokens.CacheRead != 0 || ev.Tokens.CacheCreation != 0 {
		t.Errorf("old CPA token shape changed or invented fields: %+v", ev.Tokens)
	}
}

func TestToEvent_ClaudeCPA7267AliasesKeepIndependentCacheCounters(t *testing.T) {
	raw := rawQueueItem{
		Timestamp: "2026-07-12T12:00:00+08:00",
		Provider:  "claude",
		Model:     "claude-sonnet",
		RequestID: "req-claude-cache",
		Tokens: rawTokens{
			Input: 2000, Output: 300, Cached: 1000, CacheRead: 1000, CacheCreation: 500, Total: 3800,
		},
	}
	ev, ok := toEvent(raw)
	if !ok {
		t.Fatal("expected event")
	}
	if ev.Tokens.Cached != 0 || ev.Tokens.CacheRead != 1000 || ev.Tokens.CacheCreation != 500 {
		t.Errorf("canonical Claude cache tokens = %+v, want cached=0 read=1000 creation=500", ev.Tokens)
	}
	if ev.Tokens.Input != 2000 {
		t.Errorf("Claude input = %d, want unchanged 2000", ev.Tokens.Input)
	}
}

func TestToEvent_CPA7297CanonicalAccountingComplete(t *testing.T) {
	raw := rawQueueItem{Timestamp: "2026-07-26T12:00:00+08:00", Provider: "codex", RequestID: "v2-complete",
		AccountingVersion: 2, TokenBreakdown: &rawTokenBreakdown{SchemaVersion: 2, Quality: "complete", TotalTokens: 120,
			Input:  rawInputBreakdown{TotalTokens: 100, UncachedTokens: 60, CacheReadTokens: 30, CacheWriteTokens: 10},
			Output: rawOutputBreakdown{TotalTokens: 20, NonReasoningTokens: 5, ReasoningTokens: 15}}}
	ev, ok := toEvent(raw)
	if !ok || ev.Accounting.Version != 2 || ev.Accounting.Quality != "complete" {
		t.Fatalf("canonical accounting not retained: ok=%v accounting=%+v", ok, ev.Accounting)
	}
	want := model.CanonicalTokens{UncachedInput: 60, CacheRead: 30, CacheCreation: 10, NonReasoningOutput: 5, Reasoning: 15, Total: 120}
	if ev.Accounting.Tokens != want {
		t.Fatalf("canonical tokens = %+v, want %+v", ev.Accounting.Tokens, want)
	}
}

func TestToEvent_CPA7297UnclassifiedAndInvalidFallback(t *testing.T) {
	valid := rawQueueItem{Timestamp: "2026-07-26T12:00:00+08:00", Provider: "unknown", RequestID: "v2-partial",
		AccountingVersion: 2, TokenBreakdown: &rawTokenBreakdown{SchemaVersion: 2, Quality: "unclassified", TotalTokens: 120,
			Input:  rawInputBreakdown{TotalTokens: 80, UncachedTokens: 80},
			Output: rawOutputBreakdown{TotalTokens: 20, NonReasoningTokens: 20}, UnclassifiedTokens: 20}}
	ev, ok := toEvent(valid)
	if !ok || ev.Accounting.Quality != "unclassified" || ev.Accounting.Tokens.Unclassified != 20 {
		t.Fatalf("unclassified accounting lost: ok=%v accounting=%+v", ok, ev.Accounting)
	}

	invalid := valid
	invalid.RequestID = "v2-invalid"
	invalid.Tokens = rawTokens{Input: 10, Output: 2, Total: 12}
	invalid.TokenBreakdown.Input.UncachedTokens = 81
	ev, ok = toEvent(invalid)
	if !ok || ev.Accounting.Version != 2 || ev.Accounting.Quality != "inconsistent" || ev.Accounting.Tokens.Unclassified != 120 {
		t.Fatalf("invalid v2 must be quarantined as inconsistent: ok=%v accounting=%+v", ok, ev.Accounting)
	}
}

func TestCanonicalAccountingRejectsOverflow(t *testing.T) {
	maxInt64 := int64(^uint64(0) >> 1)
	_, ok := canonicalAccounting(rawTokenBreakdown{SchemaVersion: 2, Quality: "complete", TotalTokens: maxInt64,
		Input: rawInputBreakdown{TotalTokens: maxInt64, UncachedTokens: maxInt64, CacheReadTokens: 1}})
	if ok {
		t.Fatal("overflowing canonical buckets must be rejected")
	}
}

func TestFlexString_HexStringAndNumber(t *testing.T) {
	// 回归：CPA v7.1.31 的 auth_index 是 hex hash，不能当数字解析（否则整批丢数据）
	var a flexString
	if err := a.UnmarshalJSON([]byte(`"75e9b19080b47771"`)); err != nil || a != "75e9b19080b47771" {
		t.Errorf("hex string parse: err=%v val=%q", err, a)
	}
	var b flexString
	if err := b.UnmarshalJSON([]byte(`7`)); err != nil || b != "7" {
		t.Errorf("number parse: err=%v val=%q", err, b)
	}
	var c flexString
	if err := c.UnmarshalJSON([]byte(`null`)); err != nil || c != "" {
		t.Errorf("null parse: err=%v val=%q", err, c)
	}
}
