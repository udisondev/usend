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
	ID           string
	mu           sync.RWMutex
	interactions map[string]*interaction
	signUp       func([]byte) []byte
	cluster      *cluster
	inbox        chan incomeSignal
	reactionsMu  sync.Mutex
	reactions    map[string]func(s incomeSignal) bool
	stnServer    string
	privateAuth  *ecdsa.PrivateKey
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
	ctx = span.Extend(ctx, "interactions.Run")
	logger.Debugf(ctx, "Start...")
	i.inbox = make(chan incomeSignal)

	go func() {
		<-ctx.Done()
		logger.Debugf(ctx, "...End")
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
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, member := range n.interactions {
		member.send <- s
	}
}

func (i *interactions) disconnect(ID string) {
	ctx := span.Init("interactions.disconnect <ID:%s>", ID)
	logger.Debugf(ctx, "Searchinb...")
	i.mu.Lock()
	defer i.mu.Unlock()
	member, ok := i.interactions[ID]
	if !ok {
		logger.Debugf(ctx, "Not found!")
		return
	}
	delete(i.interactions, ID)
	logger.Debugf(ctx, "Removed!")
	go member.disconnect()
}

func (n *interactions) send(ID string, s networkSignal) {
	ctx := span.Init("send to '%s'")
	logger.Debugf(
		ctx,
		"'%s' signal",
		s.Type.String(),
	)

	n.mu.RLock()
	defer n.mu.RUnlock()

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

		n.mu.Lock()
		defer n.mu.Unlock()

		delete(n.interactions, conn.ID())
	}()

	i := interaction{
		send:       out,
		disconnect: disconnect,
	}

	n.mu.Lock()
	n.interactions[conn.ID()] = &i
	n.mu.Unlock()

	connInbox := conn.Interact(ctx, out)

	go func() {
		defer disconnect()
		for in := range i.applyFilters(connInbox) {
			n.inbox <- in
		}
	}()
}

func (n *interactions) getInteraction(ID string) (*interaction, bool) {
	ctx := span.Init("interactions.getInteraction <ID:%s>", ID)
	n.mu.RLock()
	defer n.mu.RUnlock()
	logger.Debugf(ctx, "Searching...")
	memb, ok := n.interactions[ID]
	if !ok {
		logger.Warnf(ctx, "Not found!")
		return nil, false
	}
	logger.Debugf(ctx, "Found!")
	return memb, ok
}

func (n *interactions) rangeInteraction(fn func(memb *interaction)) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, memb := range n.interactions {
		fn(memb)
	}
}

func (n *interactions) compareAndSwapInteractionState(ID string, old, new interactionState) {
	n.mu.Lock()
	defer n.mu.Unlock()

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
	ctx := span.Init("interactions.memberAuthKey <ID:%s>", ID)
	i.cluster.mu.RLocker()
	defer i.cluster.mu.RUnlock()
	logger.Debugf(ctx, "Searching...")
	pubKey, ok := i.cluster.members[ID]
	if !ok {
		logger.Warnf(ctx, "Not found!")
		return nil
	}
	logger.Debugf(ctx, "Found!")
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
