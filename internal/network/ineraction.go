package network

import (
	"context"
	"slices"
	"sync"
	"time"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

type interaction struct {
	id string

	stateMu sync.RWMutex
	state   interactionState

	connectMu      sync.Mutex
	waitOffersList []string

	disconnect func()

	send chan<- networkSignal
}

func (i *interaction) addWaitOffersList(from string) {
	i.connectMu.Lock()
	defer i.connectMu.Unlock()

	i.waitOffersList = append(i.waitOffersList, from)
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
	i.stateMu.Lock()
	defer i.stateMu.Unlock()
	i.state = new
}

func (i *interaction) applyFilters(in <-chan incomeSignal) <-chan incomeSignal {
	filters := []func(in <-chan incomeSignal) <-chan incomeSignal{
		i.muteNotVerifiedFilter,
		i.offersFilter,
		i.messagesPerMinuteFilter,
	}

	out := in
	for _, filter := range filters {
		out = filter(out)
	}

	return out
}

func (n *Network) clusterBroadcast(s networkSignal) {
	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()

	for _, member := range n.interactions {
		member.send <- s
	}
}

func (n *Network) disconnect(ID string) {
	member, ok := n.interactions[ID]
	if !ok {
		return
	}
	delete(n.interactions, ID)
	member.disconnect()
}

func (n *Network) send(ID string, s networkSignal) {
	ctx := span.Init("node.Send to '%s'")
	logger.Debugf(
		ctx,
		"Going to send '%s' signal",
		s.Type.String(),
	)

	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()

	m, ok := n.interactions[ID]
	if !ok {
		logger.Debugf(
			nil,
			"Member '%s' not found",
			ID,
		)
		return
	}

	select {
	case m.send <- s:
	default:
		logger.Debugf(
			nil,
			"Disconnecting '%s' (low throuput)",
			ID,
		)
		n.disconnect(ID)
	}
}

func (n *Network) addConnection(
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

func (i *interaction) muteNotVerifiedFilter(in <-chan incomeSignal) <-chan incomeSignal {
	out := make(chan incomeSignal)

	go func() {
		defer close(out)
		for msg := range in {
			i.stateMu.RLock()
			state := i.state
			i.stateMu.RUnlock()

			if state == NotVerified {
				return
			}
			out <- msg
		}

	}()

	return out

}

func (i *interaction) offersFilter(in <-chan incomeSignal) <-chan incomeSignal {
	out := make(chan incomeSignal)

	go func() {
		defer close(out)
		for msg := range in {
			i.stateMu.RLock()
			state := i.state
			i.stateMu.RUnlock()

			if state == Connected {
				out <- msg
				continue
			}

			if msg.Type != SendOfferSignal {
				return
			}

			if !slices.Contains(i.waitOffersList, string(msg.Payload[:257])) {
				return
			}

			if i.id != string(msg.Payload[257:513]) {
				return
			}

			out <- msg
		}
	}()

	return out
}

func (i *interaction) messagesPerMinuteFilter(in <-chan incomeSignal) <-chan incomeSignal {
	out := make(chan incomeSignal)

	go func() {
		defer close(out)

		counter := 0
		for {
			select {
			case <-time.After(time.Minute):
				counter = 0
			case msg, ok := <-in:
				if !ok {
					return
				}

				if counter > maxMessagesPerMinute {
					return
				}

				counter++

				out <- msg
			}
		}
	}()

	return out
}

func (n *Network) getInteraction(ID string) (*interaction, bool) {
	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()
	memb, ok := n.interactions[ID]
	if !ok {
		return nil, false
	}
	return memb, ok
}

func (n *Network) rangeInteraction(fn func(memb *interaction)) {
	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()

	for _, memb := range n.interactions {
		fn(memb)
	}
}

func (n *Network) compareAndSwapInteractionState(ID string, old, new interactionState) {
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
