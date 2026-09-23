// Package audit 提供 append-only 的 JSONL 审计日志（按《技术架构设计》D9）。
// 约定：Event 中的 Detail 不得包含任何密码或哈希值。
package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Event struct {
	TS     time.Time `json:"ts"`
	Actor  string    `json:"actor"` // 操作人 bind DN
	IP     string    `json:"ip"`
	Op     string    `json:"op"` // login / create / modify / delete / move / import …
	DN     string    `json:"dn,omitempty"`
	Detail string    `json:"detail,omitempty"`
	Result string    `json:"result"` // ok / fail
}

type Logger struct {
	mu sync.Mutex
	f  *os.File
}

// New 打开（必要时创建）审计日志文件；path 为空时返回丢弃式 Logger（测试用）。
func New(path string) (*Logger, error) {
	if path == "" {
		return &Logger{}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &Logger{f: f}, nil
}

func (l *Logger) Log(e Event) {
	if l == nil || l.f == nil {
		return
	}
	if e.TS.IsZero() {
		e.TS = time.Now()
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.f.Write(append(b, '\n'))
}

func (l *Logger) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	return l.f.Close()
}

// ReadLast 读取审计文件最后 limit 条事件（新→旧），文件不存在返回空。
func ReadLast(path string, limit int) []Event {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	out := make([]Event, 0, limit)
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		if lines[i] == "" {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(lines[i]), &e); err == nil {
			out = append(out, e)
		}
	}
	return out
}
