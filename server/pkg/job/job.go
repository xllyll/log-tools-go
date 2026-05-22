package job

import (
	"context"
	"sync"
)

type Task func(ctx context.Context) error

type Pool struct {
	ch      chan Task
	wg      sync.WaitGroup
	workers int
}

func NewPool(workers int) *Pool {
	if workers <= 0 {
		workers = 4
	}
	p := &Pool{
		ch:      make(chan Task, workers*8),
		workers: workers,
	}
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.ch {
		_ = task(context.Background())
	}
}

func (p *Pool) Submit(task Task) {
	p.ch <- task
}

func (p *Pool) Close() {
	close(p.ch)
	p.wg.Wait()
}
