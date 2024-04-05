package zpool

import "time"

type Worker struct {
	pool *Pool
	task chan func()
	// 最后执行任务时间
	lastTime time.Time
}

func (w *Worker) run() {
	go w.running()
}

func (w *Worker) running() {
	for f := range w.task {
		if f == nil {
			return
		}
		f()
		w.pool.PutWorker(w)
		w.pool.decrRunning()
	}
}
