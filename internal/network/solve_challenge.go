package network

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

func solveChallenge(n *Network, in incomeSignal) {
	ctx := span.Init("solving challange of '%s'", in.From)
	logger.Debugf(ctx, "Start...")

	hash := sha256.Sum256(in.Payload)

	r, s, err := ecdsa.Sign(
		rand.Reader,
		n.config.privateAuth,
		hash[:],
	)
	if err != nil {
		logger.Errorf(ctx, "ecdsa.Sign: %v", err)
		return
	}

	sig := signature{R: r, S: s}
	sigBytes, err := asn1.Marshal(sig)
	if err != nil {
		logger.Errorf(ctx, "asn1.Marshal: %v", err)
		return
	}

	n.send(
		in.From,
		networkSignal{
			Type:    TestChallengeSignal,
			Payload: sigBytes,
		},
	)

	logger.Debugf(ctx, "...End")
}
