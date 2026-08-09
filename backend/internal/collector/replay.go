package collector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

const replayBufferSchemaVersion = 1

// replayBatch 是 destructive pop 后持久化的版本化批次信封。
// Items 只含可重放业务字段和脱敏 key 身份，不含明文 key、响应头或失败正文。
type replayBatch struct {
	SchemaVersion int               `json:"schema_version"`
	PoppedAt      time.Time         `json:"popped_at"`
	Items         []replayQueueItem `json:"items"`
}

// replayQueueItem 保留脱敏后的原始 CPA JSON；未知字段不丢，强类型解析延后到 durable save 之后。
type replayQueueItem struct {
	Payload           json.RawMessage `json:"payload,omitempty"`
	SanitizationError string          `json:"sanitization_error,omitempty"`
	KeyFingerprint    string          `json:"key_fingerprint"`
	KeyMask           string          `json:"key_mask"`
}

func newReplayBatchFromRaw(items []json.RawMessage, poppedAt time.Time) replayBatch {
	replayItems := make([]replayQueueItem, 0, len(items))
	for i := range items {
		replayItems = append(replayItems, sanitizeRawQueueItem(items[i]))
		items[i] = nil
	}
	return replayBatch{SchemaVersion: replayBufferSchemaVersion, PoppedAt: poppedAt, Items: replayItems}
}

func sanitizeRawQueueItem(raw json.RawMessage) replayQueueItem {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return replayQueueItem{Payload: json.RawMessage("null"), SanitizationError: fmt.Sprintf("queue item is not an object: %v", err),
			KeyFingerprint: noKeyFingerprint, KeyMask: noKeyMask}
	}
	apiKey := ""
	sanitizationError := ""
	if encoded, ok := fields["api_key"]; ok {
		if err := json.Unmarshal(encoded, &apiKey); err != nil && !bytes.Equal(bytes.TrimSpace(encoded), []byte("null")) {
			sanitizationError = "api_key must be a string or null"
		}
		delete(fields, "api_key")
	}
	delete(fields, "response_headers")
	if encoded, ok := fields["fail"]; ok {
		var fail map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fail); err == nil && fail != nil {
			delete(fail, "body")
			if sanitized, err := json.Marshal(fail); err == nil {
				fields["fail"] = sanitized
			} else {
				delete(fields, "fail")
			}
		} else {
			delete(fields, "fail")
		}
	}
	sanitized, err := json.Marshal(fields)
	if err != nil {
		return replayQueueItem{Payload: json.RawMessage("null"), SanitizationError: fmt.Sprintf("sanitize queue item: %v", err),
			KeyFingerprint: keyFingerprint(apiKey), KeyMask: keyMask(apiKey)}
	}
	return replayQueueItem{Payload: sanitized, SanitizationError: sanitizationError,
		KeyFingerprint: keyFingerprint(apiKey), KeyMask: keyMask(apiKey)}
}
