package network

import (
	"crypto/ecdsa"
	"sync"
)

type cluster struct {
	mu      sync.RWMutex
	members map[string]*ecdsa.PublicKey
}

func NewCluster() *cluster {
	return &cluster{
		members: make(map[string]*ecdsa.PublicKey),
	}
}

func (c *cluster) MemberAuthKey(ID string) *ecdsa.PublicKey {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key, ok := c.members[ID]
	if !ok {
		return nil
	}

	return key
}
