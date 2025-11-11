package xjob

import "sync"

// Job 表示一个要执行的任务
type Job struct {
	Fn     func() error // 要执行的函数（通常包含 DB 写入）
	Result chan<- error // 用于返回错误（可为 nil，表示 fire-and-forget）
}

// JobQueueManager 是单例任务队列管理器
type JobQueueManager struct {
	queue chan Job
	once  sync.Once
}

// 全局单例实例
var instance *JobQueueManager
var once sync.Once

// GetInstance 获取单例实例
func GetInstance() *JobQueueManager {
	once.Do(func() {
		instance = &JobQueueManager{
			queue: make(chan Job, 1000), // 缓冲大小可根据负载调整
		}
		go instance.start()
	})
	return instance
}

// start 启动消费者 goroutine
func (m *JobQueueManager) start() {
	for job := range m.queue {
		err := job.Fn()
		if job.Result != nil {
			job.Result <- err
		}
	}
}

// Submit 提交一个任务，并可选择是否等待结果
// 如果需要等待结果，传入 waitForResult=true；否则为 fire-and-forget
func (m *JobQueueManager) Submit(fn func() error, waitForResult bool) error {
	if !waitForResult {
		select {
		case m.queue <- Job{Fn: fn, Result: nil}:
		default:
			// 队列满时可选择丢弃或阻塞，这里简单丢弃并返回错误
			return ErrQueueFull
		}
		return nil
	}

	resultChan := make(chan error, 1)
	select {
	case m.queue <- Job{Fn: fn, Result: resultChan}:
	case <-resultChan: // 不会发生，仅为安全
	}

	return <-resultChan
}

// 可选：定义队列满的错误
var ErrQueueFull = NewQueueError("job queue is full")

type QueueError struct {
	msg string
}

func NewQueueError(msg string) error {
	return &QueueError{msg: msg}
}

func (e *QueueError) Error() string {
	return e.msg
}
