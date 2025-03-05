package network

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"udisend/pkg/crypt"
	"udisend/pkg/logger"
	"udisend/pkg/span"

	"github.com/pion/webrtc/v4"
)

func doVerify(n *Network, in incomeSignal) {
	ctx := span.Init(fmt.Sprintf("node.doVerify user=%s", in.From))
	logger.Debugf(ctx, "Start...")

	authKey := n.cluster.MemberAuthKey(in.From)
	if authKey != nil {
		logger.Debugf(ctx, "Has no member=%s in my cluster", in.From)
		return
	}

	challenge := n.challenger.Challenge(in.From, authKey)

	n.send(
		in.From,
		networkSignal{
			Type:    SolveChallengeSignal,
			Payload: challenge,
		},
	)

	logger.Debugf(ctx, "...End")
}

func newConnection(n *Network, s incomeSignal) {

}

func generateConnectionSign(n *Network, s incomeSignal) {
	recipient := string(s.Payload)
	ctx := span.Init("generateConnectionSign <Recipient:%s>", recipient)
	logger.Debugf(ctx, "Start...")
	sign := rand.Text() + rand.Text()

	n.addReaction(
		waitOfferTimeout,
		sign,
		func(nextS incomeSignal) bool {
			if nextS.Type != HandleOfferSignal {
				return false
			}
			if nextS.From != s.From {
				return false
			}
			var offer connectionOffer
			offer.unmarshal(nextS.Payload)

			if offer.From != recipient {
				return false
			}

			if offer.Sign != sign {
				return false
			}

			go handleOffer(n, offer)

			return true
		},
	)

	n.send(recipient, networkSignal{
		Type: SendConnectionSignSignal,
		Payload: connectionSign{
			To:         recipient,
			From:       n.config.id,
			Sign:       sign,
			StunServer: n.config.stunServer,
			PublicKey:  n.config.pubRsa,
		}.marshal(),
	})

	logger.Debugf(ctx, "...End")
}

func handleOffer(n *Network, s incomeSignal) {
	var offer connectionOffer
	offer.unmarshal(s.Payload)

	ctx := span.Init("handleOffer of '%s'", offer.From)
	logger.Debugf(ctx, "Start...")

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{n.config.stunServer},
			},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		logger.Errorf(ctx, "webrtc.NewPeerConnection: %v", err)
		return
	}

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateClosed {
			disconnect()
		}
		logger.Debugf(ctx, "Connection change state to '%s'", state.String())
	})

	sd := webrtc.SessionDescription{}
	sdByted, err := n.config.decode(offer.RemoteSD)
	if err != nil {
		return
	}

	err = json.Unmarshal(sdByted, &sd)
	if err != nil {
		return
	}

	err = pc.SetRemoteDescription(sd)
	if err != nil {
		logger.Errorf(ctx, "pc.SetRemoteDesctiption: %v", err)
		pc.Close()
		return
	}

	answ, err := pc.CreateAnswer(nil)
	if err != nil {
		logger.Errorf(ctx, "pc.CreateAnswer: %v", err)
		pc.Close()
		return
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)

	err = pc.SetLocalDescription(answ)
	if err != nil {
		logger.Errorf(ctx, "pc.SetLocalDescription: %v", err)
		pc.Close()
		return
	}

	<-gatherComplete

	n.send(s.From, networkSignal{
		Type: SendAnswerSignal,
		Payload: answer{
			To:       offer.From,
			From:     n.config.id,
			RemoteSD: []byte(answ.SDP),
		}.marshal(),
	})
	n.Send(message.Outcome{
		To: in.From,
		Message: message.Message{
			Type: message.SendAnswer,
			Text: strings.Join([]string{n.id, offer.From, encode(pc.LocalDescription())}, "|"),
		},
	})

	logger.Debugf(ctx, "...End")
}

func ping(n *Network, s incomeSignal) {

}

func pong(n *Network, s incomeSignal) {

}
