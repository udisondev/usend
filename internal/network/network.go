package network

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"sync"
)

type Network struct {
	config networkOpts

	interatcionsMu sync.RWMutex
	interactions   map[string]*interaction
	inbox          chan incomeSignal

	reactionsMu sync.Mutex
	reactions   map[string]func(s incomeSignal) bool

	privateKey *rsa.PrivateKey

	cluster *cluster

	challenger *challenger
}

func New(
	ID string,
	pubAuth *ecdsa.PublicKey,
	privateAuth *ecdsa.PrivateKey,
	decode func([]byte) ([]byte, error),
	pubRsa *rsa.PublicKey,
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
		interactions: make(map[string]*interaction),
		inbox:        make(chan incomeSignal),
	}

}

func (n *Network) Run(ctx context.Context) {
	n.inbox = make(chan incomeSignal)

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
