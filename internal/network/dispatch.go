package network

import (
	"udisend/model"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

var handlers = map[model.SignalType]func(*Network, model.IncomeSignal){
	model.DoVerifySignal:               doVerify,
	model.SolveChallengeSignal:         solveChallenge,
	model.TestChallengeSignal:          testChallenge,
	model.NewConnectionSignal:          newConnection,
	model.GenerateConnectionSignSignal: generateConnection,
	model.MakeOfferSignal:              makeOffer,
	model.HandleOfferSignal:            handleOffer,
	model.HandleAnswerSignal:           handleAnswer,
	model.ConnectionEstablishedSignal:  connectionEstablished,
	model.PingSignal:                   ping,
	model.PongSignal:                   pong,
}

func (n *Network) dispatch(s model.IncomeSignal) {
	ctx := span.Init("network.Dispatch")
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
