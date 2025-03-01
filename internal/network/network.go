package network

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"udisend/model"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

type Connection interface {
	ID() string
	Interact(
		ctx context.Context,
		out <-chan model.NetworkSignal,
	) <-chan model.IncomeSignal
}

type interaction struct {
	conn       Connection
	disconnect func()
	send       chan<- model.NetworkSignal
}

type Network struct {
	config         networkOpts
	interatcionsMu sync.RWMutex
	interactions   map[string]interaction
	inbox          chan model.IncomeSignal
	cluster        *cluster
	challenger     *challenger
}

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

func New(
	ID string,
	pubAuth *ecdsa.PublicKey,
	privateAuth *ecdsa.PrivateKey,
	opts ...With,
) *Network {
	cfg := networkOpts{
		id:          ID,
		pubAuth:     pubAuth,
		privateAuth: privateAuth,
	}

	for _, opt := range opts {
		cfg = opt(cfg)
	}

	return &Network{
		config:       cfg,
		cluster:      NewCluster(),
		challenger:   NewChallenger(),
		interactions: make(map[string]interaction),
		inbox:        make(chan model.IncomeSignal),
	}

}

func (n *Network) Run(ctx context.Context) {
	n.inbox = make(chan model.IncomeSignal)

	go func() {
		<-ctx.Done()
		close(n.inbox)
	}()

	for range n.config.workersNum {
		go func() {
			for s := range n.inbox {
				n.dispatch(s)
			}
		}()
	}
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

func (n *Network) addConnection(ctx context.Context, conn Connection, disconnect func()) {
	out := make(chan model.NetworkSignal)
	go func() {
		<-ctx.Done()
		close(out)
	}()

	interaction := interaction{
		conn:       conn,
		send:       out,
		disconnect: disconnect,
	}

	n.interatcionsMu.Lock()
	n.interactions[conn.ID()] = interaction
	n.interatcionsMu.Unlock()

	connInbox := conn.Interact(ctx, out)

	go func() {
		for in := range connInbox {
			n.inbox <- in
		}
	}()
}

func (n *Network) send(ID string, s model.NetworkSignal) {
	ctx := span.Init("node.Send to '%s'")
	logger.Debugf(
		ctx,
		"Going to send '%s' signal",
		s.Type.String(),
	)

	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()

	m, ok := n.interactions[ID]
	if !ok {
		logger.Debugf(
			nil,
			"Member '%s' not found",
			ID,
		)
		return
	}

	select {
	case m.send <- s:
	default:
		logger.Debugf(
			nil,
			"Disconnecting '%s' (low throuput)",
			ID,
		)
		n.disconnect(ID)
	}
}

func (n *Network) disconnect(ID string) {
	n.interatcionsMu.Lock()
	defer n.interatcionsMu.Unlock()

	member, ok := n.interactions[ID]
	if !ok {
		return
	}
	delete(n.interactions, ID)
	member.disconnect()
}
