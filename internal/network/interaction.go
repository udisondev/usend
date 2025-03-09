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
	interactionsMu sync.RWMutex
	interactions   map[string]*interaction
	signUp         func([]byte) []byte
	cluster        *cluster
	inbox          chan incomeSignal
	reactionsMu    sync.Mutex
	reactions      []*Reaction
	stnServer      string
	privateAuth    *ecdsa.PrivateKey
}

type Reaction struct {
	mu   sync.Mutex
	done bool
	fn   func(s incomeSignal) bool
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
		close(i.inbox)
		logger.Debugf(ctx, "...End")
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
	ctx := span.Init("interactions.clusterBroadcast")

	n.interactionsMu.RLock()
	logger.Debugf(ctx, "Interactions read locked")
	defer func() {
		n.interactionsMu.RUnlock()
		logger.Debugf(ctx, "Interactions read unlocked")
	}()

	logger.Debugf(ctx, "Iterating...")
	for _, member := range n.interactions {
		member.send <- s
	}
	logger.Debugf(ctx, "...Iterating completed")
}

func (i *interactions) disconnect(ID string) {
	ctx := span.Init("interactions.disconnect <ID:%s>", ID)

	i.interactionsMu.Lock()
	logger.Debugf(ctx, "Interactions locked")
	defer func() {
		i.interactionsMu.Unlock()
		logger.Debugf(ctx, "Interactions unlocked")
	}()

	logger.Debugf(ctx, "Searching...")
	member, ok := i.interactions[ID]
	if !ok {
		logger.Debugf(ctx, "Not found!")
		return
	}
	delete(i.interactions, ID)
	logger.Debugf(ctx, "Removed!")
	go member.disconnect()
}

func (i *interactions) send(ID string, s networkSignal) {
	ctx := span.Init("interactions.send <ID:%s>", ID)

	logger.Debugf(
		ctx,
		"'%s' signal",
		s.Type.String(),
	)

	i.interactionsMu.RLock()
	logger.Debugf(ctx, "Interactions read locked")
	defer func() {
		i.interactionsMu.RUnlock()
		logger.Debugf(ctx, "Interactions read unlocked")
	}()

	logger.Debugf(ctx, "Searching...")
	m, ok := i.interactions[ID]
	if !ok {
		logger.Debugf(
			nil,
			"Not found",
			ID,
		)
		return
	}

	select {
	case m.send <- s:
		logger.Debugf(ctx, "Successful sent")
	default:
		logger.Debugf(
			nil,
			"Disconnecting (low throuput)",
			ID,
		)
		go i.disconnect(ID)
	}
}

func (i *interactions) addConnection(
	ctx context.Context,
	conn connection,
	disconnect func(),
) {
	ctx = span.Extend(ctx, "interactions.addConnection <ID:%s>", conn.ID())
	out := make(chan networkSignal)
	go func() {
		<-ctx.Done()
		logger.Debugf(ctx, "Context closed!")
		close(out)

		i.interactionsMu.Lock()
		logger.Debugf(ctx, "Interactions locked")
		defer func() {
			i.interactionsMu.Unlock()
			logger.Debugf(ctx, "Interactions unlocked")
		}()

		delete(i.interactions, conn.ID())
	}()

	newI := interaction{
		send:       out,
		disconnect: disconnect,
	}

	i.interactionsMu.Lock()
	i.interactions[conn.ID()] = &newI
	i.interactionsMu.Unlock()

	connInbox := conn.Interact(ctx, out)

	go func() {
		defer disconnect()
		for in := range newI.applyFilters(connInbox) {
			i.inbox <- in
		}
	}()
}

func (n *interactions) getInteraction(ID string) (*interaction, bool) {
	ctx := span.Init("interactions.getInteraction <ID:%s>", ID)
	n.interactionsMu.RLock()
	defer n.interactionsMu.RUnlock()
	logger.Debugf(ctx, "Searching...")
	memb, ok := n.interactions[ID]
	if !ok {
		logger.Warnf(ctx, "Not found!")
		return nil, false
	}
	logger.Debugf(ctx, "Found!")
	return memb, ok
}

func (i *interactions) rangeInteractions(fn func(memb *interaction)) {
	ctx := span.Init("interactions.rangeInteractions")

	i.interactionsMu.RLock()
	logger.Debugf(ctx, "Interactions read locked")
	defer func() {
		i.interactionsMu.RUnlock()
		logger.Debugf(ctx, "Interactions read unlocked")
	}()

	logger.Debugf(ctx, "Start...")
	for _, memb := range i.interactions {
		fn(memb)
	}
	logger.Debugf(ctx, "...End")
}

func (i *interactions) compareAndSwapInteractionState(ID string, old, new interactionState) {
	ctx := span.Init("interactions.compareAndSwapInteractionState <ID:%s>", ID)

	i.interactionsMu.RLock()
	logger.Debugf(ctx, "Interactions read locked")
	defer func() {
		i.interactionsMu.RUnlock()
		logger.Debugf(ctx, "Interactions read unlocked")
	}()

	logger.Debugf(ctx, "Searching...")
	memb, ok := i.interactions[ID]
	if !ok {
		logger.Warnf(ctx, "Not found!")
		return
	}

	memb.mu.Lock()
	logger.Debugf(ctx, "Member=%s locked", memb.id)
	defer func() {
		memb.mu.Unlock()
		logger.Debugf(ctx, "Member=%s unlocked", memb.id)
	}()

	if memb.state != old {
		logger.Warnf(ctx, "Different states currendOld=%d, expectedOld=%d", memb.state, old)
		return
	}

	memb.state = new
	logger.Debugf(ctx, "State changed")
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

func (i *interactions) addReaction(timeout time.Duration, fn func(s incomeSignal) bool) {
	ctx := span.Init("interactions.addReaction")

	logger.Debugf(ctx, "Reactions locked")
	i.reactionsMu.Lock()
	defer func() {
		i.reactionsMu.Unlock()
		logger.Debugf(ctx, "Reactions unlocked")
	}()

	for idx := range i.reactions {
		if !i.reactions[idx].done {
			continue
		}
		logger.Debugf(ctx, "Re-use old reaction")
		i.reactions[idx].fn = fn
		i.reactions[idx].done = false
		go func() {
			<-time.After(timeout)
			i.reactions[idx].mu.Lock()
			defer i.reactions[idx].mu.Unlock()
			i.reactions[idx].done = true
		}()
		return
	}

	logger.Debugf(ctx, "Append reactions")
	react := &Reaction{fn: fn}
	i.reactions = append(i.reactions, react)

	go func() {
		<-time.After(timeout)
		react.mu.Lock()
		defer react.mu.Unlock()
		react.done = true
	}()
}

func (i *interactions) privateAuthKey() *ecdsa.PrivateKey {
	return i.privateAuth
}

func (i *interactions) react(s incomeSignal) {
	ctx := span.Init("interactions.react")
	logger.Debugf(ctx, "Reacts locked")
	i.reactionsMu.Lock()
	defer func() {
		i.reactionsMu.Unlock()
		logger.Debugf(ctx, "Reacts unlocked")
	}()

	logger.Debugf(ctx, "Iterate reactions...")
	for idx := range i.reactions {
		r := i.reactions[idx]
		go func(r *Reaction, in incomeSignal) {
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.done {
				return
			}
			r.done = r.fn(in)
		}(r, s)
	}
	logger.Debugf(ctx, "...Iterating completed")
}
