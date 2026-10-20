package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

// handlerCall - один вызов фейкового хендлера с аргументами.
type handlerCall struct {
	ctx context.Context
	req any
}

// fakeHandler - grpc.UnaryHandler, который записывает вызовы и отдаёт заданные resp/err.
type fakeHandler struct {
	calls []handlerCall
	resp  any
	err   error
}

var _ grpc.UnaryHandler = (&fakeHandler{}).handle

func (h *fakeHandler) handle(ctx context.Context, req any) (any, error) {
	h.calls = append(h.calls, handlerCall{ctx: ctx, req: req})
	return h.resp, h.err
}
