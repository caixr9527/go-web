package breaker

import (
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
	expiry     time.Time
}

func (cb *CircuitBreaker) NewGeneration() {
	cb.generation++
	cb.counts.Clear()
	switch cb.state {
	case Closed:
		cb.expiry = time.Now().Add(cb.interval)
	case Open:
		cb.expiry = time.Now().Add(cb.timeout)
	case HalfOpen:
		cb.expiry = time.Now()
	}
}

func NewCircuitBreaker(st Settings) *CircuitBreaker {
	cb := &CircuitBreaker{}
	cb.onStateChange = st.OnStateChange
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
	err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}
	result, err := req()
	cb.counts.OnRequest()

	err = cb.afterRequest(cb.isSuccessful(err))
	return result, err

}

func (cb *CircuitBreaker) beforeRequest() error {
	return nil
}

func (cb *CircuitBreaker) afterRequest(success bool) error {
	if success {
		cb.counts.OnSuccess()
	} else {
		cb.counts.OnFail()
	}
	return nil
}
