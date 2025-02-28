package network

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"udisend/model"
)

type Network struct {
	id             string
	publicAuth     *ecdsa.PublicKey
	privateAuth    *ecdsa.PrivateKey
	encode         func([]byte) ([]byte, error)
	decode         func([]byte) ([]byte, error)
	inbox          chan model.NetworkSignal
	interactionsMu sync.RWMutex
	interactions   map[string]interaction
}

func New(
	ID string,
	pubAuth *ecdsa.PublicKey,
	privateAuth *ecdsa.PrivateKey,
	encode func([]byte) ([]byte, error),
	decode func([]byte) ([]byte, error),
) *Network {
	return &Network{
		id:          ID,
		publicAuth:  pubAuth,
		privateAuth: privateAuth,
		encode:      encode,
		decode:      decode,
	}

}

func (n *Network) Run(ctx context.Context) {
	n.inbox = make(chan model.NetworkSignal)

	go func() {
		<-ctx.Done()
		close(n.inbox)
	}()

	go func() {
		for s := range n.inbox {
			n.dispatch(s)
		}
	}()
}
