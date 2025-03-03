package network

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"sync"
	"time"
	"udisend/model"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

type challenger struct {
	mu         sync.Mutex
	challenges map[string]challenge
}

type challenge struct {
	For    string
	Value  []byte
	PubKey *ecdsa.PublicKey
}

func NewChallenger() *challenger {
	return &challenger{
		challenges: make(map[string]challenge),
	}
}

func (c *challenger) Challenge(ID string, authKey *ecdsa.PublicKey) []byte {
	ctx := span.Init("challenger.MakeFor(%s)", ID)
	val := []byte(rand.Text())

	c.mu.Lock()
	c.challenges[ID] = challenge{
		For:    ID,
		Value:  val,
		PubKey: authKey,
	}
	c.mu.Unlock()

	go func() {
		<-time.After(5 * time.Second)
		logger.Debugf(ctx, "Removing challenge")
		c.mu.Lock()
		delete(c.challenges, ID)
		c.mu.Unlock()
	}()

	return val
}

func (c *challenger) Test(ID string, sign []byte) bool {
	ctx := span.Init("challenger.Test of '%s'", ID)

	actualChallenge, ok := c.challenges[ID]
	if !ok {
		logger.Debugf(ctx, "Has no challenge")
		return false
	}

	c.mu.Lock()
	delete(c.challenges, ID)
	c.mu.Unlock()

	var sig model.Signature
	if _, err := asn1.Unmarshal(sign, &sig); err != nil {
		logger.Debugf(ctx, "asn1.Unmarshal: %v", err)
		return false
	}

	hash := sha256.Sum256(actualChallenge.Value)
	return ecdsa.Verify(
		actualChallenge.PubKey,
		hash[:],
		sig.R,
		sig.S,
	)
}
