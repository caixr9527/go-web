package zpool

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrorInvalidCap    = errors.New("pool [cap] can not < 0")
	ErrorInvalidExpire = errors.New("pool [expire] can not < 0")
	ErrorHasClosed     = errors.New("can not submit, pool has released")
)

type sig struct {
}

const DefaultExpire = 3

type Pool struct {
	// 容量
	cap int32
	// 正在运行worker数量
	running int32
	// 空闲worker
	workers []*Worker
	// 过期时间 超过时间回收
	expire time.Duration
	// 释放资源 销毁pool
	release chan sig

	lock sync.Mutex
	// 只能调用一次
	once sync.Once
}

func NewPool(cap int32) (*Pool, error) {
	return NewPoolWithExpire(cap, DefaultExpire)
}

func NewPoolWithExpire(cap int32, expire int32) (*Pool, error) {
	if cap <= 0 {
		return nil, ErrorInvalidCap
	}
	if expire <= 0 {
		return nil, ErrorInvalidExpire
	}
	pool := &Pool{
		cap:     cap,
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
	}
	go pool.expireWorker()
	return pool, nil
}

func (p *Pool) expireWorker() {
	ticker := time.NewTicker(p.expire)
	for range ticker.C {
		if p.IsRelease() {
			break
		}
		p.lock.Lock()
		workers := p.workers
		n := len(workers) - 1
		if n >= 0 {
			for i, w := range workers {
				if time.Now().Sub(w.lastTime) <= p.expire {
					break
				}
				n = i
				w.task <- nil
			}
			if n >= len(workers)-1 {
				p.workers = workers[:0]
			} else {
				p.workers = workers[n+1:]
			}
			fmt.Printf("清除完成，running:%d, workers:%v \n", p.running, p.workers)
		}
		p.lock.Unlock()
	}
}

func (p *Pool) Submit(task func()) error {
	if len(p.release) > 0 {
		return ErrorHasClosed
	}
	w := p.GetWorker()
	w.task <- task
	w.pool.incrRunning()
	return nil
}

func (p *Pool) GetWorker() *Worker {
	// 获取pool里面的worker
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	// 没有空闲，则需要新建一个
	// 有正在运行的worker + 空闲的 > cap，阻塞等待worker释放
	if n >= 0 {
		p.lock.Lock()
		w := idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
		p.lock.Unlock()
		return w
	}
	if p.running < p.cap {
		// 有空闲
		w := &Worker{
			pool: p,
			task: make(chan func(), 1),
		}
		w.run()
		return w
	}
	for {
		p.lock.Lock()
		idleWorkers := p.workers
		n := len(idleWorkers) - 1
		if n < 0 {
			p.lock.Unlock()
			continue
		}

		w := idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
		p.lock.Unlock()
		return w
	}
}

func (p *Pool) incrRunning() {
	atomic.AddInt32(&p.running, 1)
}

func (p *Pool) PutWorker(w *Worker) {
	w.lastTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.lock.Unlock()
}

func (p *Pool) decrRunning() {
	atomic.AddInt32(&p.running, -1)
}

func (p *Pool) Release() {
	p.once.Do(func() {
		p.lock.Lock()
		workers := p.workers
		for i, w := range workers {
			w.task = nil
			w.pool = nil
			workers[i] = nil
		}
		p.workers = nil
		p.lock.Unlock()
		p.release <- sig{}
	})
}

func (p *Pool) Restart() bool {
	if len(p.release) <= 0 {
		return true
	}
	_ = <-p.release
	go p.expireWorker()
	return true
}

func (p *Pool) IsRelease() bool {
	return len(p.release) > 0
}
