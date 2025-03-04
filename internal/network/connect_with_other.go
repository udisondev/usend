package network

import (
	"context"
	"crypto/rand"
	"time"
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
	n.reactions[sendSignReactionKey] = func(s incomeSignal) bool {
		if s.Type != SendConnectionSignSignal {
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
				n.send(ID, networkSignal{
					Type:    MakeOfferSignal,
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
	n.reactions[connectionsEstablishedReactKey] = func(s incomeSignal) bool {
		if s.Type != ConnectionEstablishedSignal {
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

		user.send <- networkSignal{
			Type:    GenerateConnectionSignSignal,
			Payload: []byte(ID),
		}
	}

	go func() {
		select {
		case <-time.After(waitingConnectionEstablishingTimeout):
			n.dropReaction(sendSignReactionKey)
			n.dropReaction(connectionsEstablishedReactKey)

			n.disconnect(ID)

			n.clusterBroadcast(networkSignal{
				Type:    DisconnectCandidate,
				Payload: []byte(ID),
			})
		case <-connectionsEstablishedCtx.Done():
			n.interatcionsMu.RLock()
			defer n.interatcionsMu.RUnlock()

			member.setState(Connected)
		}
	}()
}
