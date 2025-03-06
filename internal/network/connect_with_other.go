package network

import (
	"context"
	"crypto/rand"
	"time"
)

func (n *Network) connectWithOther(ID string) {
	member, ok := n.getInteraction(ID)
	if !ok {
		return
	}

	var confirmedConnections int
	var signsProvided int

	signsAreReadyCtx, signsAreReady := context.WithCancel(context.Background())

	reqConns := min(minNetworkConns, len(n.cluster.members))

	n.addReaction(waitingSignTimeout,
		rand.Text(),
		func(s incomeSignal) bool {
			if s.Type != SendConnectionSignSignal {
				return false
			}
			if ID != string(s.Payload[:idLength]) {
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
		})

	connectionsEstablishedCtx, connectionsEstablished := context.WithCancel(context.Background())
	n.addReaction(
		waitingConnectionEstablishingTimeout,
		rand.Text(),
		func(s incomeSignal) bool {
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
		})

	n.rangeInteraction(func(memb *interaction) {
		if memb.id == ID {
			return
		}

		memb.send <- networkSignal{
			Type:    GenerateConnectionSignSignal,
			Payload: []byte(ID),
		}
	})

	go func() {
		select {
		case <-time.After(waitingConnectionEstablishingTimeout):
			n.disconnect(ID)
			n.clusterBroadcast(networkSignal{
				Type:    DisconnectCandidate,
				Payload: []byte(ID),
			})
		case <-connectionsEstablishedCtx.Done():
			n.compareAndSwapInteractionState(ID, NotConnected, Connected)
		}
	}()
}
