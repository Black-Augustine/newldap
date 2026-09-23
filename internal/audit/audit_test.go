package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendAndReadBack(t *testing.T) {
	f := filepath.Join(t.TempDir(), "log", "audit.jsonl")
	l, err := New(f)
	if err != nil {
		t.Fatal(err)
	}
	l.Log(Event{Actor: "cn=admin,dc=example,dc=cn", IP: "10.8.0.15", Op: "modify", DN: "uid=zhangwei,ou=tech,dc=example,dc=cn", Result: "ok"})
	l.Log(Event{Actor: "uid=zhangwei,ou=tech,dc=example,dc=cn", Op: "login", Result: "fail"})
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	r, err := os.Open(f)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var events []Event
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		var e Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("审计行不是合法 JSON: %v", err)
		}
		if e.TS.IsZero() {
			t.Fatal("时间戳未被自动填充")
		}
		events = append(events, e)
	}
	if len(events) != 2 {
		t.Fatalf("应记录 2 条事件，得到 %d", len(events))
	}
	if events[0].Op != "modify" || events[1].Result != "fail" {
		t.Errorf("事件内容回读不一致: %+v", events)
	}
}

func TestDiscardLogger(t *testing.T) {
	l, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	l.Log(Event{Op: "x"}) // 不应 panic
}
