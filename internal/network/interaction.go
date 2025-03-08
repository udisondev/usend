package network

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"time"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

type interactions struct {
	ID             string
	interatcionsMu sync.RWMutex
	interactions   map[string]*interaction
	signUp         func([]byte) []byte
	cluster        *cluster
	inbox          chan incomeSignal
	reactionsMu    sync.Mutex
	reactions      map[string]func(s incomeSignal) bool
	stnServer      string
	privateAuth    *ecdsa.PrivateKey
}

type interaction struct {
	id         string
	mu         sync.RWMutex
	state      interactionState
	decode     func(b []byte) ([]byte, error)
	encode     func(b []byte) ([]byte, error)
	disconnect func()
	send       chan<- networkSignal
}

func (i *interactions) Run(ctx context.Context, countOfWorkers int) {
	i.inbox = make(chan incomeSignal)

	go func() {
		<-ctx.Done()
		close(i.inbox)
	}()

	for range countOfWorkers {
		go func() {
			for s := range i.inbox {
				i.dispatch(s)
			}
		}()
	}

}

type connection interface {
	ID() string
	Interact(
		ctx context.Context,
		out <-chan networkSignal,
	) <-chan incomeSignal
}

type interactionState uint8

const (
	NotVerified interactionState = iota
	NotConnected
	Connected
	Disconnected
)

func (i *interaction) setState(new interactionState) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.state = new
}

func (i *interaction) applyFilters(in <-chan incomeSignal) <-chan incomeSignal {
	filters := []func(in <-chan incomeSignal) <-chan incomeSignal{
		i.muteNotVerifiedFilter,
		i.messagesPerMinuteFilter,
	}

	out := in
	for _, filter := range filters {
		out = filter(out)
	}

	return out
}

func (n *interactions) clusterBroadcast(s networkSignal) {
	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()

	for _, member := range n.interactions {
		member.send <- s
	}
}

func (n *interactions) disconnect(ID string) {
	member, ok := n.interactions[ID]
	if !ok {
		return
	}
	delete(n.interactions, ID)
	member.disconnect()
}

func (n *interactions) send(ID string, s networkSignal) {
	ctx := span.Init("send to '%s'")
	logger.Debugf(
		ctx,
		"'%s' signal",
		s.Type.String(),
	)

	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()

	m, ok := n.interactions[ID]
	if !ok {
		logger.Debugf(
			nil,
			"Member not found",
			ID,
		)
		return
	}

	select {
	case m.send <- s:
	default:
		logger.Debugf(
			nil,
			"Disconnecting (low throuput)",
			ID,
		)
		n.disconnect(ID)
	}
}

func (n *interactions) addConnection(
	ctx context.Context,
	conn connection,
	disconnect func(),
) {
	out := make(chan networkSignal)
	go func() {
		<-ctx.Done()
		close(out)

		n.interatcionsMu.Lock()
		defer n.interatcionsMu.Unlock()

		delete(n.interactions, conn.ID())
	}()

	i := interaction{
		send:       out,
		disconnect: disconnect,
	}

	n.interatcionsMu.Lock()
	n.interactions[conn.ID()] = &i
	n.interatcionsMu.Unlock()

	connInbox := conn.Interact(ctx, out)

	go func() {
		defer disconnect()
		for in := range i.applyFilters(connInbox) {
			n.inbox <- in
		}
	}()
}

func (n *interactions) getInteraction(ID string) (*interaction, bool) {
	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()
	memb, ok := n.interactions[ID]
	if !ok {
		return nil, false
	}
	return memb, ok
}

func (n *interactions) rangeInteraction(fn func(memb *interaction)) {
	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()

	for _, memb := range n.interactions {
		fn(memb)
	}
}

func (n *interactions) compareAndSwapInteractionState(ID string, old, new interactionState) {
	n.interatcionsMu.Lock()
	defer n.interatcionsMu.Unlock()

	memb, ok := n.interactions[ID]
	if !ok {
		return
	}
	if memb.state != old {
		return
	}
	memb.state = new
}

func (i *interactions) clusterSize() int {
	return len(i.cluster.members)
}

func (i *interactions) memberAuthKey(ID string) *ecdsa.PublicKey {
	i.cluster.mu.RLocker()
	defer i.cluster.mu.RUnlock()

	pubKey, ok := i.cluster.members[ID]
	if !ok {
		return nil
	}
	return pubKey
}

func (i *interactions) myID() string {
	return i.ID
}

func (i *interactions) stunServer() string {
	return i.stnServer
}

func (i *interactions) addReaction(timeout time.Duration, key string, fn func(s incomeSignal) bool) {
	i.reactionsMu.Lock()
	defer i.reactionsMu.Unlock()

	i.reactions[key] = fn

	go func() {
		<-time.After(timeout)
		i.reactionsMu.Lock()
		defer i.reactionsMu.Unlock()
		delete(i.reactions, key)
	}()
}

func (i *interactions) privateAuthKey() *ecdsa.PrivateKey {
	return i.privateAuth
}
