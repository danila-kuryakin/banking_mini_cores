package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

// handlerCall - один вызов фейкового хендлера с аргументами
type handlerCall struct {
	ctx context.Context
	req any
}

// fakeHandler - фейковый grpc.UnaryHandler: записывает вызовы и отдаёт заданные resp и err
type fakeHandler struct {
	resp  any
	err   error
	calls []handlerCall
}

func (h *fakeHandler) handle(ctx context.Context, req any) (any, error) {
	h.calls = append(h.calls, handlerCall{ctx: ctx, req: req})
	return h.resp, h.err
}

var _ grpc.UnaryHandler = (&fakeHandler{}).handle
