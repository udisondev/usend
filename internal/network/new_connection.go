package network

import (
	"context"
	"udisend/model"
)

type Connection interface {
	ID() string
	Interact(ctx context.Context, out <-chan model.NetworkSignal) <-chan model.NetworkSignal
	Disconnect()
}

type interaction struct {
	conn Connection
	send chan<- model.NetworkSignal
}

func (n *Network) AddConnection(ctx context.Context, conn Connection) {
	out := make(chan model.NetworkSignal)
	go func() {
		<-ctx.Done()
		close(out)
	}()

	interaction := interaction{
		conn: conn,
		send: out,
	}

	n.interactionsMu.Lock()
	n.interactions[conn.ID()] = interaction
	n.interactionsMu.Unlock()

	connInbox := conn.Interact(ctx, out)

	go func() {
		for in := range connInbox {
			n.inbox <- in
		}
	}()
}
