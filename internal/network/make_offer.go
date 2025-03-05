package network

import (
	"encoding/json"
	"strings"
	"udisend/pkg/crypt"
	"udisend/pkg/logger"
	"udisend/pkg/span"

	"github.com/pion/webrtc/v4"
)

func makeOffer(n *Network, s incomeSignal) {
	var connSign connectionSign
	err := connSign.unmarshal(s.Payload)
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
		logger.Errorf(ctx, "webrtc.NewPeerConnection <stubServer:%s>: %v", connSign.StunServer, err)
		return
	}

	dc, err := pc.CreateDataChannel("private", nil)
	if err != nil {
		logger.Errorf(ctx, "pc.CreateDataChannel <stubServer:%s>: %v", connSign.StunServer, err)
		pc.Close()
		return
	}

	of, err := pc.CreateOffer(nil)
	if err != nil {
		dc.Close()
		pc.Close()
		return
	}

	if err := pc.SetLocalDescription(of); err != nil {
		logger.Errorf(ctx, "pc.SetLocalDescription: %v", err)
		dc.Close()
		pc.Close()
		return
	}

	n.reactionsMu.Lock()
	defer n.reactionsMu.Unlock()

	n.addReaction(
		waitRTCAnswer,
		connSign.Sign,
		func(nextS incomeSignal) bool {
			if nextS.Type != HandleAnswerSignal {
				return false
			}
			if nextS.From != s.From {
				return false
			}
			var answ answer
			answ.unmarshal(nextS.Payload)

			if answ.To != n.config.id {
				return false
			}
			if answ.From != connSign.From {
				return false
			}

			remoteSD, err := crypt.DecryptMessage(answ.RemoteSD, n.privateKey)
			if err != nil {
				return false
			}

			var sess webrtc.SessionDescription
			err = json.Unmarshal(remoteSD, &sess)
			if err != nil {
				return false
			}

			err = pc.SetRemoteDescription(sess)
			if err != nil {
				return false
			}

			return true
		},
	)

	n.send(s.From, networkSignal{
		Type:    SendOfferSignal,
		Payload: offer{}.marshal(),
	})

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
