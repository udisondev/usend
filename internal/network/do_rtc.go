package network

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"udisend/pkg/crypt"
	"udisend/pkg/logger"
	"udisend/pkg/span"

	"github.com/pion/webrtc/v4"
)

type offerer struct {
	privateRSA *rsa.PrivateKey
	pubRSA     *rsa.PublicKey
	offer      rtcOffer
}

type answerer struct {
	privateRSA *rsa.PrivateKey
	pubRSA     *rsa.PublicKey
	answer     rtcAnswer
}

func makeOffer(n dispatcher, s incomeSignal) {
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

	privateKey, publicKey, err := crypt.GenerateRSAKeys()
	if err != nil {
		return
	}
	n.addReaction(
		waitRTCAnswer,
		connSign.Sign,
		func(nextS incomeSignal) bool {
			if nextS.Type != SignalTypeHandleAnswer {
				return false
			}
			if nextS.From != s.From {
				return false
			}
			var answ rtcAnswer
			answ.unmarshal(nextS.Payload)

			if answ.To != n.myID() {
				return false
			}
			if answ.From != connSign.From {
				return false
			}

			remoteSD, err := crypt.DecryptMessage(answ.RemoteSD, privateKey)
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

	encrypted, err := crypt.EncryptMessage([]byte(of.SDP), connSign.PubKey)
	if err != nil {
		return
	}

	outBytes, err := rtcOffer{
		To:       connSign.From,
		From:     n.myID(),
		Sign:     connSign.Sign,
		PubKey:   publicKey,
		RemoteSD: encrypted,
	}.marshal()
	if err != nil {
		return
	}

	n.send(s.From, networkSignal{
		Type:    SignalTypeSendOffer,
		Payload: outBytes,
	})

	logger.Debugf(ctx, "...End")
}

func generateConnectionSign(n dispatcher, s incomeSignal) {
	recipient := string(s.Payload)
	ctx := span.Init("generateConnectionSign <Recipient:%s>", recipient)
	logger.Debugf(ctx, "Start...")
	sign := rand.Text() + rand.Text()

	private, public, err := crypt.GenerateRSAKeys()
	if err != nil {
		return
	}

	n.addReaction(
		waitOfferTimeout,
		sign,
		func(nextS incomeSignal) bool {
			if nextS.Type != SignalTypeHandleOffer {
				return false
			}
			if nextS.From != s.From {
				return false
			}
			var offer rtcOffer
			offer.unmarshal(nextS.Payload)

			if offer.From != recipient {
				return false
			}

			if offer.Sign != sign {
				return false
			}

			go handleOffer(n, offerer{
				privateRSA: private,
				pubRSA:     public,
				offer:      offer,
			})

			return true
		},
	)

	payload, err := connectionSign{
		To:         recipient,
		From:       n.myID(),
		Sign:       sign,
		StunServer: n.stunServer(),
		PubKey:     public,
	}.marshal()
	if err != nil {
		return
	}
	n.send(recipient, networkSignal{
		Type:    SignalTypeSendConnectionSign,
		Payload: payload,
	})

	logger.Debugf(ctx, "...End")
}

func handleOffer(n dispatcher, c offerer) {
	ctx := span.Init("handleOffer of '%s'", c.offer.From)
	logger.Debugf(ctx, "Start...")

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{n.stunServer()},
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
			//			disconnect()
		}
		logger.Debugf(ctx, "Connection change state to '%s'", state.String())
	})

	sd := webrtc.SessionDescription{}
	sdByted, err := crypt.DecryptMessage(c.offer.RemoteSD, c.privateRSA)
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

	n.send(c.offer.From, networkSignal{
		Type: SignalTypeSendAnswer,
		Payload: rtcAnswer{
			To:       c.offer.From,
			From:     n.myID(),
			RemoteSD: []byte(answ.SDP),
		}.marshal(),
	})

	logger.Debugf(ctx, "...End")
}
