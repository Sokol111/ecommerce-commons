package health

import "sync"

type Readiness interface {
	AddOne()
	IsReady() bool
	Done()
}

type readiness struct {
	wg    *sync.WaitGroup
	once  sync.Once
	ready chan struct{}
}

func newReadiness() *readiness {
	return &readiness{
		wg:    &sync.WaitGroup{},
		ready: make(chan struct{}),
	}
}

func (r *readiness) StartWatching() {
	go func() {
		r.once.Do(func() {
			r.wg.Wait()
			close(r.ready)
		})
	}()
}

func (r *readiness) AddOne() {
	r.wg.Add(1)
}

func (r *readiness) IsReady() bool {
	select {
	case <-r.ready:
		return true
	default:
		return false
	}
}

func (r *readiness) Done() {
	r.wg.Done()
}
