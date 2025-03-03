package network

import (
	"context"
	"crypto/rand"
	"time"
	"udisend/model"
)

func (n *Network) connectWithOther(ID string) {
	n.interatcionsMu.RLock()
	defer n.interatcionsMu.RUnlock()

	member, ok := n.interactions[ID]
	if !ok {
		return
	}

	var confirmedConnections int
	var signsProvided int

	n.reactionsMu.Lock()
	defer n.reactionsMu.Unlock()

	signsAreReadyCtx, signsAreReady := context.WithCancel(context.Background())

	reqConns := min(minNetworkConns, len(n.cluster.members))

	sendSignReactionKey := rand.Text()
	n.reactions[sendSignReactionKey] = func(s model.IncomeSignal) bool {
		if s.Type != model.SendConnectionSignSignal {
			return false
		}
		if ID != string(s.Payload[:257]) {
			return false
		}

		signsProvided++
		go func() {
			select {
			case <-time.After(waitingSignTimeout):
			case <-signsAreReadyCtx.Done():
				member.addWaitOffersList(s.From)
				n.send(ID, model.NetworkSignal{
					Type:    model.MakeOfferSignal,
					Payload: s.Payload,
				})
			}
		}()

		if signsProvided == reqConns {
			signsAreReady()
			return true
		}

		return false
	}

	connectionsEstablishedReactKey := rand.Text()
	connectionsEstablishedCtx, connectionsEstablished := context.WithCancel(context.Background())
	n.reactions[connectionsEstablishedReactKey] = func(s model.IncomeSignal) bool {
		if s.Type != model.ConnectionEstablishedSignal {
			return false
		}
		if ID != string(s.Payload) {
			return false
		}

		confirmedConnections++
		if confirmedConnections == reqConns {
			connectionsEstablished()
			return true
		}

		return false
	}

	for id, user := range n.interactions {
		if id == ID {
			continue
		}

		user.send <- model.NetworkSignal{
			Type:    model.GenerateConnectionSignSignal,
			Payload: []byte(ID),
		}
	}

	go func() {
		select {
		case <-time.After(waitingConnectionEstablishingTimeout):
			n.dropReaction(sendSignReactionKey)
			n.dropReaction(connectionsEstablishedReactKey)

			n.disconnect(ID)

			n.clusterBroadcast(model.NetworkSignal{
				Type:    model.DisconnectCandidate,
				Payload: []byte(ID),
			})
		case <-connectionsEstablishedCtx.Done():
			n.interatcionsMu.RLock()
			defer n.interatcionsMu.RUnlock()

			member.setState(Connected)
		}
	}()
}
