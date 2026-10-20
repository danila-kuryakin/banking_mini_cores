package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

// fakeHandler - заглушка grpc.UnaryHandler, записывающая вызовы
type fakeHandler struct {
	resp   any
	err    error
	calls  int
	gotCtx context.Context
	gotReq any
}

var _ grpc.UnaryHandler = (&fakeHandler{}).handle

func (h *fakeHandler) handle(ctx context.Context, req any) (any, error) {
	h.calls++
	h.gotCtx = ctx
	h.gotReq = req
	return h.resp, h.err
}
