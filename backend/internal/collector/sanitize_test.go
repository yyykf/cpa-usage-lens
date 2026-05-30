package collector

import "testing"

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
