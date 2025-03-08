package network

import (
	"context"
	"crypto/rand"
	"sync/atomic"
	"time"
	"udisend/pkg/logger"
	"udisend/pkg/span"
)

func connectWithOther(d dispatcher, ID string) {
	ctx := span.Init("connectWithOther <ID:%s>", ID)
	var confirmedConnections atomic.Int32
	var signsProvided atomic.Int32

	signsAreReadyCtx, signsAreReady := context.WithCancel(context.Background())

	reqConns := min(minNetworkConns, d.clusterSize())
	logger.Debugf(ctx, "Required %d connections", reqConns)

	d.addReaction(waitingSignTimeout,
		rand.Text(),
		func(nextS incomeSignal) bool {
			if nextS.Type != SignalTypeSendConnectionSign {
				return false
			}
			if ID != string(nextS.Payload[:idLength]) {
				return false
			}
			ctx := span.Init("waitingSign for=%s", ID)
			logger.Debugf(ctx, "Received sign from=%s", nextS.From)
			signsProvided.Add(1)
			go func() {
				select {
				case <-time.After(waitingSignTimeout):
					logger.Warnf(ctx, "Timeout!")
				case <-signsAreReadyCtx.Done():
					logger.Debugf(ctx, "Going to send sign from=%s", nextS.From)
					d.send(ID, networkSignal{
						Type:    SignalTypeMakeOffer,
						Payload: nextS.Payload,
					})
				}
			}()

			if int(signsProvided.Load()) == reqConns {
				logger.Debugf(ctx, "All required signs collected!")
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
			if s.Type != SignalTypeConnectionEstablished {
				return false
			}
			if ID != string(s.Payload) {
				return false
			}

			logger.Debugf(nil, "%s established connection with %s", s.From, ID)
			confirmedConnections.Add(1)
			if int(confirmedConnections.Load()) == reqConns {
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
			Type:    SignalTypeGenerateConnectionSign,
			Payload: []byte(ID),
		}
	})

	go func() {
		ctx := span.Init("waiting connection establishing of %s", ID)
		select {
		case <-time.After(waitingConnectionEstablishingTimeout):
			logger.Warnf(ctx, "Timeout!")
			d.disconnect(ID)
			go d.clusterBroadcast(networkSignal{
				Type:    SignalTypeDisconnectCandidate,
				Payload: []byte(ID),
			})
		case <-connectionsEstablishedCtx.Done():
			logger.Debugf(ctx, "All connections established!")
			d.compareAndSwapInteractionState(ID, NotConnected, Connected)
		}
	}()
}
