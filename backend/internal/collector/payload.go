package collector

import "encoding/json"

// rawQueueItem 是 CPA GET /v0/management/usage-queue 返回的单条原始 payload。
// 含敏感/大字段（api_key、response_headers、fail.body），仅用于解析；
// 由 toEvent 转成精简明细时丢弃这些字段，绝不入库。
type rawQueueItem struct {
	Timestamp       string          `json:"timestamp"`
	LatencyMs       *int32          `json:"latency_ms"`
	TTFTMs          *int32          `json:"ttft_ms"`
	Source          string          `json:"source"`
	AuthIndex       flexString      `json:"auth_index"`
	Tokens          rawTokens       `json:"tokens"`
	Failed          bool            `json:"failed"`
	Fail            *rawFail        `json:"fail"`
	Provider        string          `json:"provider"`
	Model           string          `json:"model"`
	Alias           string          `json:"alias"`
	Endpoint        string          `json:"endpoint"`
	AuthType        string          `json:"auth_type"`
	APIKey          string          `json:"api_key"`          // 敏感：剥离，不入库
	RequestID       string          `json:"request_id"`
	ReasoningEffort string          `json:"reasoning_effort"`
	ServiceTier     string          `json:"service_tier"`
	ResponseHeaders json.RawMessage `json:"response_headers"` // 大+敏感：剥离，不入库
}

// rawTokens 字段顺序/类型与 model.Tokens 完全一致，便于直接类型转换。
type rawTokens struct {
	Input         int64 `json:"input_tokens"`
	Output        int64 `json:"output_tokens"`
	Reasoning     int64 `json:"reasoning_tokens"`
	Cached        int64 `json:"cached_tokens"`
	CacheRead     int64 `json:"cache_read_tokens"`
	CacheCreation int64 `json:"cache_creation_tokens"`
	Total         int64 `json:"total_tokens"`
}

type rawFail struct {
	StatusCode *int32 `json:"status_code"`
	Body       string `json:"body"` // 剥离，不入库
}

// flexString 容忍 auth_index 是 JSON string（CPA v7.1.31 实测为 hex hash 如 "75e9b19080b47771"）
// 或 number，统一转成字符串——用字符串才不会因一条解析失败而丢掉整批已 pop 的数据。
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(b) == 0 || s == "null" {
		return nil
	}
	if b[0] == '"' {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		*f = flexString(str)
		return nil
	}
	*f = flexString(s) // number 等非字符串：保留原文
	return nil
}
