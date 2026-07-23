package guard

import (
	"testing"
	"time"
)

func TestCurrentMemoryMB(t *testing.T) {
	mem := CurrentMemoryMB()
	if mem <= 0 {
		t.Fatalf("expected positive memory, got %.1f", mem)
	}
	t.Logf("current memory: %.1fMB", mem)
}

func TestMemoryReport(t *testing.T) {
	report := MemoryReport()
	if report == "" {
		t.Fatalf("expected non-empty report")
	}
	t.Logf("memory report: %s", report)
}

func TestForceGC(t *testing.T) {
	// 不 panic 即通过
	ForceGC()
}

func TestMemoryGuardTriggersOnExceed(t *testing.T) {
	// 替换重启函数为空操作，避免测试进程退出
	origRestart := restartFn
	restartFn = func() {}
	defer func() { restartFn = origRestart }()

	triggered := make(chan struct{}, 1)

	// 设一个极低的上限（1MB），几乎必然超限
	g := New(1, 1, func() {
		select {
		case triggered <- struct{}{}:
		default:
		}
	})
	g.Start()
	defer g.Stop()

	select {
	case <-triggered:
		// 成功触发
	case <-time.After(5 * time.Second):
		t.Fatalf("guard did not trigger within timeout")
	}
}
