package nacosx

import "testing"

// Nacos 未启用(Host 为空)时 MustRegister 不连接任何服务,返回可安全调用的 no-op 注销函数
func TestMustRegisterDisabledIsNoop(t *testing.T) {
	deregister := MustRegister(Config{}, "user-rpc", "0.0.0.0:9000")
	if deregister == nil {
		t.Fatal("deregister func must not be nil")
	}
	deregister()
}

func TestParseListenOn(t *testing.T) {
	cases := map[string]int{"0.0.0.0:9000": 9000, ":9001": 9001, " 127.0.0.1:9002 ": 9002}
	for in, want := range cases {
		got, err := ParseListenOn(in)
		if err != nil || got != want {
			t.Errorf("ParseListenOn(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"9000", "host:abc", ":0"} {
		if _, err := ParseListenOn(bad); err == nil {
			t.Errorf("ParseListenOn(%q) should fail", bad)
		}
	}
}
