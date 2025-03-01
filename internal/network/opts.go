package network

import (
	"crypto/ecdsa"
	"crypto/rsa"
)

type networkOpts struct {
	id          string
	listenAddr  string
	entryPoint  string
	workersNum  int
	decode      func([]byte) ([]byte, error)
	pubRsa      *rsa.PublicKey
	pubAuth     *ecdsa.PublicKey
	privateAuth *ecdsa.PrivateKey
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
