package network

import (
	"encoding/json"
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

	encrypted, err := crypt.EncryptMessage([]byte(of.SDP), connSign.PublicKey)
	if err != nil {
		return
	}

	outBytes, err := connectionOffer{
		To:        connSign.From,
		From:      n.config.id,
		Sign:      connSign.Sign,
		PublicKey: n.config.pubRsa,
		RemoteSD:  encrypted,
	}.marshal()
	if err != nil {
		return
	}

	n.send(s.From, networkSignal{
		Type:    SendOfferSignal,
		Payload: outBytes,
	})

	logger.Debugf(ctx, "...End")
}
