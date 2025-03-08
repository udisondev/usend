package network

import (
	"bytes"
	"crypto/rsa"
	"math/big"
	"udisend/pkg/crypt"
)

/*
ENUM(

	DoVerify,
	ProvidePubKey,
	PubKeyProvided,
	SolveChallenge,
	TestChallenge,
	NewConnection,
	GenerateConnectionSign,
	SendConnectionSign,
	MakeOffer,
	SendOffer,
	HandleOffer,
	SendAnswer,
	HandleAnswer,
	ConnectionEstablished,
	Ping,
	Pong,
	DisconnectCandidate,

)
*/
type signalType string

type networkSignal struct {
	Type    signalType
	Payload []byte
}

type incomeSignal struct {
	From string
	networkSignal
}

type signature struct {
	R, S *big.Int
}

type connectionSign struct {
	To, From, Sign, StunServer string
	PubKey                     *rsa.PublicKey
}

type rtcOffer struct {
	To, From, Sign string
	PubKey         *rsa.PublicKey
	RemoteSD       []byte
}

type rtcAnswer struct {
	To, From string
	RemoteSD []byte
}

func (o rtcOffer) marshal() ([]byte, error) {
	pubKeyBytes, err := crypt.MarshalPublicKey(o.PubKey)
	if err != nil {
		return nil, err
	}
	totalLen := len(pubKeyBytes) + len(o.To) + len(o.From) + len(o.RemoteSD) + 1
	out := make([]byte, 0, totalLen)
	out = append(out, []byte(o.To)...)
	out = append(out, []byte(o.From)...)
	out = append(out, []byte(o.Sign)...)
	out = append(out, o.RemoteSD...)
	out = append(out, '|')
	out = append(out, pubKeyBytes...)

	return out, nil
}

func (o *rtcOffer) unmarshal(b []byte) error {
	minLen := idLength*2 + signLength + pubKeyLength
	if len(b) < minLen {
		return ErrInvalidMessage
	}
	del := bytes.Index(b, []byte{'|'})
	if del < minLen-pubKeyLength {
		return ErrInvalidMessage
	}
	if len(b[del:]) != pubKeyLength {
		return ErrInvalidMessage
	}
	pubKey, err := crypt.ParsePublicKey(b[del:])
	if err != nil {
		return err
	}
	o.PubKey = pubKey
	pos := 0
	o.To = string(b[pos:idLength])
	pos += idLength
	o.From = string(b[pos : pos+idLength])
	pos += idLength
	o.Sign = string(b[pos : pos+signLength])
	pos += signLength
	o.RemoteSD = b[pos:del]
	return nil
}

func (a rtcAnswer) marshal() []byte {
	totalLen := len(a.To) + len(a.From) + len(a.RemoteSD)
	out := make([]byte, 0, totalLen)
	out = append(out, []byte(a.To)...)
	out = append(out, []byte(a.From)...)
	out = append(out, a.RemoteSD...)
	return out
}

func (a *rtcAnswer) unmarshal(b []byte) error {
	if len(b) < idLength*2 {
		return ErrInvalidMessage
	}
	pos := 0
	a.To = string(b[pos:idLength])
	pos += idLength
	a.From = string(b[pos : pos+idLength])
	pos += idLength
	a.RemoteSD = b[pos:]
	return nil
}

func (c connectionSign) marshal() ([]byte, error) {
	pubKeyBytes, err := crypt.MarshalPublicKey(c.PubKey)
	if err != nil {
		return nil, err
	}
	totalLen := idLength*2 + signLength + len(c.StunServer) + 1 + len(pubKeyBytes)
	out := make([]byte, 0, totalLen)
	out = append(out, []byte(c.To)...)
	out = append(out, []byte(c.From)...)
	out = append(out, []byte(c.Sign)...)
	out = append(out, []byte(c.StunServer)...)
	out = append(out, '|')
	out = append(out, pubKeyBytes...)
	return out, nil
}

func (c *connectionSign) unmarshal(b []byte) error {
	minLen := idLength*2 + signLength + pubKeyLength
	if len(b) < minLen {
		return ErrInvalidMessage
	}
	del := bytes.Index(b, []byte{'|'})
	if del < idLength*2+signLength {
		return ErrInvalidMessage
	}
	if len(b[del:]) != pubKeyLength {
		return ErrInvalidMessage
	}
	pubKey, err := crypt.ParsePublicKey(b[del:])
	if err != nil {
		return err
	}
	c.PubKey = pubKey
	pos := 0
	c.To = string(b[pos:idLength])
	pos += idLength
	c.From = string(b[pos : pos+idLength])
	pos += idLength
	c.Sign = string(b[pos : pos+signLength])
	pos += signLength
	c.StunServer = string(b[pos:del])
	return nil
}
