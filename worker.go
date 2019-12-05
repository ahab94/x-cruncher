package xcruncher

import (
	"context"
)

type worker struct {
	ID      int
	ctx     context.Context
	counter *counter
	stop    chan struct{}
	work    chan Executable
	pool    chan chan Executable
}

// newWorker - initializes a new worker
func newWorker(ctx context.Context, id int, pool chan chan Executable, counter *counter) *worker {
	return &worker{
		ID:      id,
		ctx:     ctx,
		pool:    pool,
		counter: counter,
		work:    make(chan Executable),
		stop:    make(chan struct{}),
	}
}

// Start - readies worker for execution
func (w *worker) Start() {
	log(w.ctx).Debugf("worker [%d] is starting...", w.ID)
	go w.do()
}

// Stop - stops the worker routine
func (w *worker) Stop() {
	close(w.stop)
}

func (w *worker) execute(exec Executable) {
	if !exec.IsCompleted() {
		defer recoverPanic(w.ctx, exec)
		defer w.counter.Done()
		w.counter.Add()
		if err := exec.Execute(); err != nil {
			log(w.ctx).Errorf("worker [%d]: error while executing: %v", w.ID, exec)
			exec.OnFailure(err)
			return
		}
		log(w.ctx).Infof("worker [%d]: completed executed: %v", w.ID, exec)
		exec.OnSuccess()
	}
}

func (w *worker) do() {
	for {
		select {
		case w.pool <- w.work:
			log(w.ctx).Debugf("worker [%d] back in queue...", w.ID)
		case exec := <-w.work:
			log(w.ctx).Debugf("worker [%d] executing %v...", w.ID, exec)
			w.execute(exec)
		case <-w.stop:
			log(w.ctx).Debugf("worker [%d] stopping...", w.ID)
			return
		}
	}
}
