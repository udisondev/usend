package network

import (
	"context"
	"crypto/rand"
	"time"
)

func connectWithOther(d dispatcher, ID string) {
	var confirmedConnections int
	var signsProvided int

	signsAreReadyCtx, signsAreReady := context.WithCancel(context.Background())

	reqConns := min(minNetworkConns, d.clusterSize())

	d.addReaction(waitingSignTimeout,
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
					d.send(ID, networkSignal{
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
	d.addReaction(
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

	d.rangeInteraction(func(memb *interaction) {
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
			d.disconnect(ID)
			d.clusterBroadcast(networkSignal{
				Type:    DisconnectCandidate,
				Payload: []byte(ID),
			})
		case <-connectionsEstablishedCtx.Done():
			d.compareAndSwapInteractionState(ID, NotConnected, Connected)
		}
	}()
}
