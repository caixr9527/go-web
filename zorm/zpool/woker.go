package zpool

import (
	zormLog "github.com/caixr9527/zorm/log"
	"time"
)

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
	defer func() {
		w.pool.decrRunning()
		w.pool.workerCache.Put(w)
		if err := recover(); err != nil {
			if w.pool.PanicHandler != nil {
				w.pool.PanicHandler()
			} else {
				zormLog.Default().Error(err)
			}
		}
		w.pool.cond.Signal()
	}()
	for f := range w.task {
		if f == nil {
			w.pool.workerCache.Put(w)
			return
		}
		f()
		w.pool.PutWorker(w)
		w.pool.decrRunning()
	}
}
