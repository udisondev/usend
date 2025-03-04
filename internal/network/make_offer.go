package network

import (
	"strings"
	"time"
	"udisend/pkg/logger"
	"udisend/pkg/span"

	"github.com/pion/webrtc/v4"
)

func makeOffer(n *Network, s incomeSignal) {
	connSign, err := unmarshalConnectionSign(s.Payload)
	if err != nil {
		return
	}

	ctx := span.Init("node.makeOffer for '%s'", connSign.From)
	logger.Debugf(ctx, "Start...")

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{connSign.StunServer},
			},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		logger.Errorf(ctx, "webrtc.NewPeerConnection <stubServer:%s>: %v", connSign.Stun, err)
		return
	}

	dc, err := pc.CreateDataChannel("private", nil)
	if err != nil {
		logger.Errorf(ctx, "pc.CreateDataChannel <stubServer:%s>: %v", connSign.Stun, err)
		pc.Close()
		return
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		dc.Close()
		pc.Close()
		return
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		logger.Errorf(ctx, "pc.SetLocalDescription: %v", err)
		dc.Close()
		pc.Close()
		return
	}

	answered := member.NewAnswerICE(connSign.From, dc, pc)
	n.waitAnswerMu.Lock()
	n.waitAnswer[connSign.From] = answered
	n.waitAnswerMu.Unlock()

	go func() {
		<-time.After(10 * time.Second)
		logger.Debugf(ctx, "No wait answer yet")

		n.waitAnswerMu.Lock()
		defer n.waitAnswerMu.Unlock()

		_, ok := n.waitAnswer[connSign.From]
		if ok {
			delete(n.waitAnswer, connSign.From)
			dc.Close()
			pc.Close()
		}
	}()

	n.Send(message.Outcome{
		To: in.From,
		Message: message.Message{
			Type: message.SendOffer,
			Text: strings.Join([]string{
				n.id,
				connSign.From,
				connSign.Sign,
				connSign.Stun,
				encode(pc.LocalDescription()),
			}, "|"),
		},
	})

	logger.Debugf(ctx, "...End")
}
