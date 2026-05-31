package config

import "testing"

func TestGetenvBool(t *testing.T) {
	const key = "TEST_COLLECTOR_ENABLED"
	cases := []struct {
		name string
		set  bool   // 是否设置环境变量
		val  string // 环境变量值
		def  bool   // 默认值
		want bool
	}{
		{name: "true", set: true, val: "true", def: false, want: true},
		{name: "false", set: true, val: "false", def: true, want: false},
		{name: "empty falls back to def", set: false, def: true, want: true},
		{name: "invalid falls back to def", set: true, val: "notabool", def: true, want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.set {
				t.Setenv(key, c.val)
			}
			if got := getenvBool(key, c.def); got != c.want {
				t.Errorf("getenvBool(%q=%q, def=%v) = %v, want %v", key, c.val, c.def, got, c.want)
			}
		})
	}
}
