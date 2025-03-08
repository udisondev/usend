package network

import (
	"crypto/ecdsa"
	"time"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

type clusterKeeper interface {
	clusterSize() int
	memberAuthKey(ID string) *ecdsa.PublicKey
}

type interactor interface {
	addReaction(timeout time.Duration, key string, fn func(s incomeSignal) bool)
	getInteraction(ID string) (*interaction, bool)
	rangeInteraction(fn func(memb *interaction))
	send(ID string, msg networkSignal)
	disconnect(ID string)
	clusterBroadcast(networkSignal)
	compareAndSwapInteractionState(ID string, old, new interactionState)
}

type dispatcher interface {
	clusterKeeper
	interactor
	privateAuthKey() *ecdsa.PrivateKey
	myID() string
	stunServer() string
}

var handlers = map[signalType]func(dispatcher, incomeSignal){
	SignalTypeDoVerify:               sendChallenge,
	SignalTypeSolveChallenge:         solveChallenge,
	SignalTypeGenerateConnectionSign: generateConnectionSign,
	SignalTypeMakeOffer:              makeOffer,
}

func (n *interactions) dispatch(s incomeSignal) {
	ctx := span.Init("interactions.dispatch")
	logger.Debugf(ctx, "Received signal <From:%s> <Type:%s>", s.From, s.Type.String())

	n.reactionsMu.Lock()
	logger.Debugf(ctx, "Reactions locked")
	for k, r := range n.reactions {
		if r(s) {
			delete(n.reactions, k)
		}
	}
	n.reactionsMu.Unlock()
	logger.Debugf(ctx, "Reactions unlocked")

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
