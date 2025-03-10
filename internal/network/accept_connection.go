package network

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"time"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

func sendChallenge(d dispatcher, in incomeSignal) {
	ctx := span.Init("sendChallenge <Recipient:%s>", in.From)
	logger.Debugf(ctx, "Start...")

	challenge := []byte(rand.Text() + rand.Text())

	d.addReaction(3*time.Second,
		func(nextIn incomeSignal) bool {
			if nextIn.From != in.From {
				return false
			}
			if nextIn.Type != SignalTypeTestChallenge {
				return false
			}

			ctx := span.Init("testChallenge of '%s'", in.From)
			logger.Debugf(ctx, "Start...")

			var sig signature
			if _, err := asn1.Unmarshal(nextIn.Payload, &sig); err != nil {
				logger.Errorf(ctx, "asn1.Unmarshal: %v", err)
				return true
			}

			pubAuth := d.memberAuthKey(nextIn.From)

			hash := sha256.Sum256(challenge)
			if !ecdsa.Verify(
				pubAuth,
				hash[:],
				sig.R,
				sig.S,
			) {
				logger.Warnf(ctx, "Failed!")
				return true
			}

			logger.Debugf(ctx, "Success!")

			d.compareAndSwapInteractionState(in.From, NotVerified, NotConnected)
			go connectWithOther(d, in.From)
			logger.Debugf(ctx, "...End")
			return true
		})

	d.send(
		in.From,
		networkSignal{
			Type:    SignalTypeSolveChallenge,
			Payload: challenge,
		},
	)

	logger.Debugf(ctx, "...End")
}

func solveChallenge(n dispatcher, in incomeSignal) {
	ctx := span.Init("solving challange of '%s'", in.From)
	logger.Debugf(ctx, "Start...")

	hash := sha256.Sum256(in.Payload)

	r, s, err := ecdsa.Sign(
		rand.Reader,
		n.privateAuthKey(),
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
			Type:    SignalTypeTestChallenge,
			Payload: sigBytes,
		},
	)

	logger.Debugf(ctx, "...End")
}
