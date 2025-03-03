package network

import (
	"fmt"
	"udisend/model"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

func doVerify(n *Network, in model.IncomeSignal) {
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
		model.NetworkSignal{
			Type:    model.SolveChallengeSignal,
			Payload: challenge,
		},
	)

	logger.Debugf(ctx, "...End")
}

func newConnection(n *Network, s model.IncomeSignal) {

}

func generateConnection(n *Network, s model.IncomeSignal) {

}

func makeOffer(n *Network, s model.IncomeSignal) {

}

func handleOffer(n *Network, s model.IncomeSignal) {

}

func handleAnswer(n *Network, s model.IncomeSignal) {

}

func connectionEstablished(n *Network, s model.IncomeSignal) {

}

func ping(n *Network, s model.IncomeSignal) {

}

func pong(n *Network, s model.IncomeSignal) {

}
