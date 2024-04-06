package zpool

import (
	"math"
	"runtime"
	"sync"
	"testing"
	"time"
)

var (
	MiB = 1024 * 1024
)

var (
	Param    = 100
	PoolSize = 1000
	TestSize = 10000
	n        = 800000
)

var curMem uint64

const (
	RunTimes           = 1000000
	BenchParam         = 10
	DefaultExpiredTime = 10 * time.Second
)

func demoFunc() {
	time.Sleep(time.Duration(BenchParam) * time.Millisecond)
}

func TestNoPool(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			demoFunc()
			wg.Done()
		}()
	}
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	//curMem = mem.TotalAlloc/MiB - curMem
	curMem = mem.TotalAlloc/(1024*1024) - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestHasPool(t *testing.T) {
	pool, _ := NewPool(math.MaxInt32)
	defer pool.Release()
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = pool.Submit(func() {
			demoFunc()
			wg.Done()
		})
	}
	wg.Wait()

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	//curMem = mem.TotalAlloc/MiB - curMem
	curMem = mem.TotalAlloc/(1024*1024) - curMem
	t.Logf("memory usage:%d MB", curMem)
	t.Logf("running worker:%d", pool.Running())
	t.Logf("free worker:%d", pool.Free())
}
