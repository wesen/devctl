package runtime

import (
	"sync"

	"github.com/go-go-golems/devctl/pkg/protocol"
	"github.com/pkg/errors"
)

type router struct {
	mu      sync.Mutex
	pending map[string]chan protocol.Response
}

func newRouter() *router {
	return &router{pending: map[string]chan protocol.Response{}}
}

func (r *router) register(rid string) chan protocol.Response {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan protocol.Response, 1)
	r.pending[rid] = ch
	return ch
}

func (r *router) deliver(rid string, resp protocol.Response) {
	r.mu.Lock()
	ch, ok := r.pending[rid]
	if ok {
		delete(r.pending, rid)
	}
	r.mu.Unlock()
	if ok {
		ch <- resp
	}
}

func (r *router) cancel(rid string) {
	r.mu.Lock()
	ch, ok := r.pending[rid]
	if ok {
		delete(r.pending, rid)
	}
	r.mu.Unlock()
	if ok {
		close(ch)
	}
}

func (r *router) failAll(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for rid, ch := range r.pending {
		delete(r.pending, rid)
		ch <- protocol.Response{
			Type:      protocol.FrameResponse,
			RequestID: rid,
			Ok:        false,
			Error: &protocol.Error{
				Code:    "E_RUNTIME",
				Message: errors.Wrap(err, "runtime").Error(),
			},
		}
	}
}
