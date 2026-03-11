package event

import (
	"context"

	"github.com/FelixSeptem/baymax/core/types"
)

type Dispatcher struct {
	handlers []types.EventHandler
}

func NewDispatcher(handlers ...types.EventHandler) *Dispatcher {
	return &Dispatcher{handlers: handlers}
}

func (d *Dispatcher) Emit(ctx context.Context, ev types.Event) {
	for _, h := range d.handlers {
		h.OnEvent(ctx, ev)
	}
}
