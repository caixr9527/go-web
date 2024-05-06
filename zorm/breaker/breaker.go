package breaker

import (
	"errors"
	"sync"
	"time"
)

type Stat int

const (
	Closed Stat = iota
	HalfOpen
	Open
)

type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailure         uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

func (c *Counts) OnRequest() {
	c.Requests++
}

func (c *Counts) OnSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *Counts) OnFail() {
	c.TotalFailure++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

func (c *Counts) Clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailure = 0
	c.ConsecutiveFailures = 0
	c.ConsecutiveSuccesses = 0
}

type Settings struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from Stat, to Stat)
	IsSuccessful  func(err error) bool
	Fallback      func(err error) (any, error)
}

type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	isSuccessful  func(err error) bool
	onStateChange func(name string, from Stat, to Stat)

	mutex      sync.Mutex
	state      Stat
	generation uint64
	counts     Counts
	expiry     time.Time // todo 时间设置有问题
	Fallback   func(err error) (any, error)
}

func (cb *CircuitBreaker) NewGeneration() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.generation++
	cb.counts.Clear()
	var zero time.Time
	switch cb.state {
	case Closed:
		if cb.interval == 0 {
			cb.expiry = zero

		} else {
			cb.expiry = time.Now().Add(cb.interval)
		}
	case Open:
		cb.expiry = time.Now().Add(cb.timeout)
	case HalfOpen:
		cb.expiry = zero
	}
}

func NewCircuitBreaker(st Settings) *CircuitBreaker {
	cb := &CircuitBreaker{}
	cb.onStateChange = st.OnStateChange
	cb.Fallback = st.Fallback
	if st.MaxRequests == 0 {
		cb.maxRequests = 1
	} else {
		cb.maxRequests = st.MaxRequests
	}
	if st.Interval == 0 {
		cb.interval = time.Duration(0) * time.Second
	} else {
		cb.interval = st.Interval
	}
	if st.Timeout == 0 {
		st.Timeout = time.Duration(20) * time.Second
	} else {
		cb.timeout = st.Timeout
	}
	if st.ReadyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	} else {
		cb.readyToTrip = st.ReadyToTrip
	}
	if st.IsSuccessful == nil {
		cb.isSuccessful = func(err error) bool {
			return err == nil
		}
	} else {
		cb.isSuccessful = st.IsSuccessful
	}
	cb.NewGeneration()
	return cb
}

func (cb *CircuitBreaker) Execute(req func() (any, error)) (any, error) {
	err, generation := cb.beforeRequest()
	if err != nil {
		// 降级
		if cb.Fallback != nil {
			return cb.Fallback(err)
		}
		return nil, err
	}
	result, err := req()
	cb.counts.OnRequest()

	cb.afterRequest(generation, cb.isSuccessful(err))
	return result, err

}

func (cb *CircuitBreaker) beforeRequest() (error, uint64) {
	now := time.Now()
	state, generation := cb.currentState(now)
	if state == Open {
		// todo
		return errors.New("断路器为打开状态"), generation
	}
	if state == HalfOpen {
		if cb.counts.Requests > cb.maxRequests {
			// todo
			return errors.New("请求数量过多"), generation
		}
	}
	return nil, generation
}

func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}
	if success {
		cb.OnSuccess(state)
	} else {
		cb.OnFail(state)
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) (Stat, uint64) {
	switch cb.state {
	case Closed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.NewGeneration()
		}
	case Open:
		if cb.expiry.Before(now) {
			cb.SetState(HalfOpen)
		}
	}
	return cb.state, cb.generation
}

func (cb *CircuitBreaker) SetState(target Stat) {
	if cb.state == target {
		return
	}
	before := cb.state
	cb.state = target
	cb.NewGeneration()
	if cb.onStateChange == nil {
		cb.onStateChange(cb.name, before, target)
	}
}

func (cb *CircuitBreaker) OnSuccess(state Stat) {
	switch state {
	case Closed:
		cb.counts.OnSuccess()
	case HalfOpen:
		cb.counts.OnSuccess()
		if cb.counts.ConsecutiveSuccesses > cb.maxRequests {
			cb.SetState(Closed)
		}
	}
}

func (cb *CircuitBreaker) OnFail(state Stat) {
	switch state {
	case Closed:
		cb.counts.OnFail()
		if cb.readyToTrip(cb.counts) {
			cb.SetState(Open)
		}
	case HalfOpen:
		cb.counts.OnFail()
		if cb.readyToTrip(cb.counts) {
			cb.SetState(Open)
		}
	}
}
