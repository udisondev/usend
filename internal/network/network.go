package network

import (
	"context"
	"crypto/ecdsa"
	"udisend/pkg/span"
)

type Network struct {
	config networkOpts
	interactions
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
		config: cfg,
		interactions: interactions{
			interactions: make(map[string]*interaction),
		},
	}

}

func (n *Network) Run(ctx context.Context) {
	ctx = span.Extend(ctx, "network.Run")
	n.interactions.Run(ctx, n.config.workersNum)
}
