package network

import (
	"udisend/model"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

var handlers = map[model.SignalType]func(*Network, model.NetworkSignal) error{
	model.DoVerify: DoVerify,
}

func (n *Network) dispatch(sig model.NetworkSignal) {
	ctx := span.Init("network.Dispatch")
	h, ok := handlers[sig.Type]
	if !ok {
		logger.Debugf(ctx, "Has no suittable handler for '%s' type", sig.Type.String())
		return
	}

	err := h(n, sig)
	if err != nil {
		logger.Debugf(ctx, "Error handle '%s' signal", sig.Type.String())
	}
}
