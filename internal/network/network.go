package network

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"sync"
	"udisend/model"
)

type networkOpts struct {
	id          string
	listenAddr  string
	entryPoint  string
	decode      func([]byte) ([]byte, error)
	pubRsa      *rsa.PublicKey
	pubAuth     *ecdsa.PublicKey
	privateAuth *ecdsa.PrivateKey
}

type Network struct {
	inbox          chan model.NetworkSignal
	interactionsMu sync.RWMutex
	interactions   map[string]interaction
	config         networkOpts
}

type With func(networkOpts) networkOpts

func WithListenAddr(v string) With {
	return func(o networkOpts) networkOpts {
		o.listenAddr = v
		return o
	}
}

func WithEntypoint(v string) With {
	return func(o networkOpts) networkOpts {
		o.entryPoint = v
		return o
	}
}

func WithCrypt(decode func([]byte) ([]byte, error), pubRSA *rsa.PublicKey) With {
	return func(o networkOpts) networkOpts {
		o.decode = decode
		o.pubRsa = pubRSA
		return o
	}
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
		inbox:        make(chan model.NetworkSignal),
		interactions: make(map[string]interaction),
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
