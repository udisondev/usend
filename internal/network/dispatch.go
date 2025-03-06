package network

import (
	"time"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

var handlers = map[signalType]func(*Network, incomeSignal){
	DoVerifySignal:               doVerify,
	SolveChallengeSignal:         solveChallenge,
	NewConnectionSignal:          newConnection,
	GenerateConnectionSignSignal: generateConnectionSign,
	MakeOfferSignal:              makeOffer,
	PingSignal:                   ping,
}

func (n *Network) dispatch(s incomeSignal) {
	ctx := span.Init("network.Dispatch")

	n.reactionsMu.Lock()
	for k, r := range n.reactions {
		if r(s) {
			delete(n.reactions, k)
		}
	}
	n.reactionsMu.Unlock()

	h, ok := handlers[s.Type]
	if !ok {
		logger.Debugf(
			ctx,
			"Has no suittable handler for '%s' type",
			s.Type.String(),
		)
		return
	}

	h(n, s)
}

func (n *Network) dropReaction(key string) {
	n.reactionsMu.Lock()
	defer n.reactionsMu.Unlock()
	delete(n.reactions, key)
}

func (n *Network) addReaction(timeout time.Duration, key string, fn func(s incomeSignal) bool) {
	n.reactionsMu.Lock()
	defer n.reactionsMu.Unlock()

	n.reactions[key] = fn

	go func() {
		<-time.After(timeout)
		n.dropReaction(key)
	}()
}
