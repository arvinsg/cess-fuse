package fs

import (
	"sync"
)

type Ticket struct {
	Total uint32

	total       uint32
	outstanding uint32

	mu   sync.Mutex
	cond *sync.Cond
}

func (t Ticket) Init() *Ticket {
	t.cond = sync.NewCond(&t.mu)
	t.total = t.Total
	return &t
}

func (t *Ticket) Take(howmany uint32, block bool) (took bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for t.outstanding+howmany > t.total {
		if block {
			t.cond.Wait()
		} else {
			return
		}
	}

	t.outstanding += howmany
	took = true
	return
}

func (t *Ticket) Return(howmany uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.outstanding -= howmany
	t.cond.Signal()
}
