package network

import (
	"fmt"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

func doVerify(n *Network, in incomeSignal) {
	ctx := span.Init(fmt.Sprintf("node.doVerify user=%s", in.From))
	logger.Debugf(ctx, "Start...")

	authKey := n.cluster.MemberAuthKey(in.From)
	if authKey != nil {
		logger.Debugf(ctx, "Has no member=%s in my cluster", in.From)
		return
	}

	challenge := n.challenger.Challenge(in.From, authKey)

	n.send(
		in.From,
		networkSignal{
			Type:    SolveChallengeSignal,
			Payload: challenge,
		},
	)

	logger.Debugf(ctx, "...End")
}

func newConnection(n *Network, s incomeSignal) {

}

func generateConnection(n *Network, s incomeSignal) {

}

func handleOffer(n *Network, s incomeSignal) {

}

func handleAnswer(n *Network, s incomeSignal) {

}

func connectionEstablished(n *Network, s incomeSignal) {

}

func ping(n *Network, s incomeSignal) {

}

func pong(n *Network, s incomeSignal) {

}
