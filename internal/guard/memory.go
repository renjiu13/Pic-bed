// Package guard 内存守护：监控进程内存占用，超限时优雅关闭并自动重启。
// 专为弱内存设备（玩客云、树莓派等）设计，防止大图压缩导致 OOM。
package guard

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// MemoryGuard 内存守护器
type MemoryGuard struct {
	limitMB     int           // 内存上限（MB）
	interval    time.Duration // 检查间隔
	onExceed    func()        // 超限回调（通常是优雅关闭服务）
	mu          sync.Mutex
	stopped     bool
	stopCh      chan struct{}
}

// New 创建内存守护器
// limitMB: 内存上限，超过则触发重启
// intervalSec: 检查间隔（秒）
// onExceed: 超限时回调，用于优雅关闭 HTTP 服务和释放资源
func New(limitMB, intervalSec int, onExceed func()) *MemoryGuard {
	if limitMB <= 0 {
		limitMB = 200
	}
	if intervalSec <= 0 {
		intervalSec = 30
	}
	return &MemoryGuard{
		limitMB:  limitMB,
		interval: time.Duration(intervalSec) * time.Second,
		onExceed: onExceed,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动内存监控
func (g *MemoryGuard) Start() {
	go g.watch()
	log.Printf("[guard] 内存守护已启动: 上限 %dMB, 检查间隔 %v", g.limitMB, g.interval)
}

// Stop 停止监控
func (g *MemoryGuard) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stopped {
		return
	}
	g.stopped = true
	close(g.stopCh)
}

// restartFn 可替换的重启函数（测试时注入空操作避免退出进程）
var restartFn = restart

func (g *MemoryGuard) watch() {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	for {
		select {
		case <-g.stopCh:
			return
		case <-ticker.C:
			used := CurrentMemoryMB()
			if used > float64(g.limitMB) {
				log.Printf("[guard] ⚠️ 内存超限! 当前 %.1fMB > 上限 %dMB，触发重启", used, g.limitMB)
				// 先触发优雅关闭回调
				if g.onExceed != nil {
					g.onExceed()
				}
				// 执行重启
				restartFn()
			}
		}
	}
}

// CurrentMemoryMB 获取当前进程内存占用（MB）
func CurrentMemoryMB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Sys) / 1024 / 1024
}

// restart 重启当前进程
// Linux: syscall.Exec 原地替换进程（PID 不变，systemd 友好）
// Windows/其他: 启动新进程后退出旧进程
func restart() {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("[guard] 获取可执行路径失败: %v，直接退出（依赖外部重启）", err)
		os.Exit(1)
	}

	// Linux: 原地替换进程（最可靠，无端口冲突）
	if runtime.GOOS == "linux" {
		log.Printf("[guard] Linux 原地重启: %s %v", exe, os.Args[1:])
		err := syscall.Exec(exe, os.Args, os.Environ())
		if err != nil {
			log.Printf("[guard] syscall.Exec 失败: %v，回退到启动新进程", err)
		}
	}

	// 通用方案：启动新进程，旧进程退出
	log.Printf("[guard] 启动新进程: %s %v", exe, os.Args[1:])
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		log.Printf("[guard] 启动新进程失败: %v，直接退出（依赖外部重启）", err)
		os.Exit(1)
	}

	// 新进程已启动，旧进程退出
	log.Printf("[guard] 新进程 PID=%d 已启动，旧进程退出", cmd.Process.Pid)
	os.Exit(0)
}

// ForceGC 强制垃圾回收（供外部在低内存时主动调用）
func ForceGC() {
	runtime.GC()
	runtime.GC() // 两次，确保彻底回收
}

// MemoryReport 返回内存报告字符串
func MemoryReport() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("Sys=%.1fMB Heap=%.1fMB Stack=%.1fMB",
		float64(m.Sys)/1024/1024,
		float64(m.HeapAlloc)/1024/1024,
		float64(m.StackInuse)/1024/1024,
	)
}
