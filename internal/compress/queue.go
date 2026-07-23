package compress

import (
	"log"
	"sync"
)

// Task 压缩任务
type Task struct {
	InputPath string
	Cfg       Config
	// OnDone 回调，压缩完成后调用（无论成功/跳过/失败）
	// resultPath 为最终文件路径（可能是 webp 或原图），err 为错误
	OnDone func(resultPath string, err error)
}

// Queue 异步压缩队列
// 单 worker 串行处理，避免弱设备并发压缩导致内存飙升
type Queue struct {
	tasks chan Task
	stop  chan struct{}
	done  chan struct{}
	wg    sync.WaitGroup
}

// NewQueue 创建压缩队列
// bufferSize 为队列缓冲大小，超出时丢弃最旧任务（保护弱设备不被积压压垮）
func NewQueue(bufferSize int) *Queue {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return &Queue{
		tasks: make(chan Task, bufferSize),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
}

// Start 启动 worker
func (q *Queue) Start() {
	q.wg.Add(1)
	go q.worker()
}

// Stop 优雅停止（等待当前任务完成）
func (q *Queue) Stop() {
	close(q.stop)
	q.wg.Wait()
}

// Enqueue 入队一个压缩任务
// 返回 false 表示队列已满被丢弃
func (q *Queue) Enqueue(task Task) bool {
	select {
	case q.tasks <- task:
		return true
	default:
		// 队列满，尝试丢弃最旧任务腾位置
		select {
		case <-q.tasks:
			log.Printf("[compress-queue] 队列满，丢弃旧任务")
		default:
		}
		select {
		case q.tasks <- task:
			return true
		default:
			log.Printf("[compress-queue] 队列仍满，丢弃新任务: %s", task.InputPath)
			return false
		}
	}
}

// worker 串行处理压缩任务
func (q *Queue) worker() {
	defer q.wg.Done()
	defer close(q.done)

	for {
		select {
		case <-q.stop:
			// 停止前处理完队列中剩余任务
			q.drainRemaining()
			return
		case task := <-q.tasks:
			q.process(task)
		}
	}
}

// drainRemaining 处理停止时队列中剩余任务
func (q *Queue) drainRemaining() {
	for {
		select {
		case task := <-q.tasks:
			q.process(task)
		default:
			return
		}
	}
}

// process 处理单个压缩任务
func (q *Queue) process(task Task) {
	resultPath, err := CompressToTarget(task.InputPath, task.Cfg)
	if err != nil {
		log.Printf("[compress-queue] 压缩失败 %s: %v", task.InputPath, err)
	} else if resultPath != task.InputPath {
		log.Printf("[compress-queue] 压缩完成 %s -> %s", task.InputPath, resultPath)
	}

	if task.OnDone != nil {
		task.OnDone(resultPath, err)
	}
}

// Pending 获取当前队列中待处理任务数
func (q *Queue) Pending() int {
	return len(q.tasks)
}
