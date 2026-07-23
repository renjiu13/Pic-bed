package compress

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueueProcessesTasks(t *testing.T) {
	q := NewQueue(10)
	q.Start()
	defer q.Stop()

	var processed int32
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		done := make(chan struct{})
		// 不实际压缩（EnableCompression=false），验证队列回调
		ok := q.Enqueue(Task{
			InputPath: "test.png",
			Cfg:       Config{EnableCompression: false},
			OnDone: func(resultPath string, err error) {
				atomic.AddInt32(&processed, 1)
				close(done)
				wg.Done()
			},
		})
		if !ok {
			t.Fatalf("enqueue failed")
		}
		go func() {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				wg.Done()
			}
		}()
	}

	wg.Wait()
	if got := atomic.LoadInt32(&processed); got != 5 {
		t.Fatalf("expected 5 processed, got %d", got)
	}
}

func TestQueueStopsGracefully(t *testing.T) {
	q := NewQueue(10)
	q.Start()

	// 入队几个任务
	for i := 0; i < 3; i++ {
		q.Enqueue(Task{
			InputPath: "test.png",
			Cfg:       Config{EnableCompression: false},
		})
	}

	// Stop 应等待任务完成
	done := make(chan struct{})
	go func() {
		q.Stop()
		close(done)
	}()

	select {
	case <-done:
		// 成功停止
	case <-time.After(10 * time.Second):
		t.Fatalf("queue stop timed out")
	}
}
