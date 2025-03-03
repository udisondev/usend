package network

import (
	"udisend/model"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

func testChallenge(n *Network, in model.IncomeSignal) {
	ctx := span.Init("testChallenge of '%s'", in.From)
	logger.Debugf(ctx, "Start...")

	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()
	member, ok := n.interactions[in.From]
	if !ok {
		return
	}

	success := n.challenger.Test(in.From, in.Payload)
	if !success {
		logger.Debugf(ctx, "challenge failed")
		member.disconnect()
		return
	}
	logger.Debugf(ctx, "challenge successful pass")

	member.setState(NotConnected)

	go n.connectWithOther(in.From)
	logger.Debugf(ctx, "...End")
}
