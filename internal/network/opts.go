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
	privateKey  *rsa.PrivateKey
	pubAuth     *ecdsa.PublicKey
	privateAuth *ecdsa.PrivateKey
	stunServer  string
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

func WithStunServer(v string) With {
	return func(o networkOpts) networkOpts {
		o.stunServer = v
		return o
	}
}
