package network

import (
	"crypto/rsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"udisend/pkg/crypt"
)

type signalType uint8

const (
	DoVerifySignal               signalType = 0x00
	ProvidePubKeySignal                     = 0x01
	PubKeyProvidedSignal                    = 0x02
	SolveChallengeSignal                    = 0x03
	TestChallengeSignal                     = 0x04
	NewConnectionSignal                     = 0x05
	GenerateConnectionSignSignal            = 0x06
	SendConnectionSignSignal                = 0x07
	MakeOfferSignal                         = 0x08
	SendOfferSignal                         = 0x09
	HandleOfferSignal                       = 0x0A
	SendAnswerSignal                        = 0x0B
	HandleAnswerSignal                      = 0x0C
	ConnectionEstablishedSignal             = 0x0D
	PingSignal                              = 0x0E
	PongSignal                              = 0x0F
	DisconnectCandidate                     = 0x10
)

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
	PublicKey                  *rsa.PublicKey
}

type offer struct {
	To, From, Sign string
	RemoteSD       []byte
}

type answer struct {
	To, From string
	RemoteSD []byte
}

func (o offer) marshal() []byte {
	return slices.Concat([]byte(o.To), []byte(o.From), []byte(o.Sign), o.RemoteSD)
}

func (o *offer) unmarshal(b []byte) {
	pos := 0
	o.To = string(b[:idLength])
	pos += idLength

	o.From = string(b[pos : pos+idLength])
	pos += idLength

	o.Sign = string(b[pos : pos+signLength])
	pos += signLength

	o.RemoteSD = b[pos:]
}

func (a *answer) unmarshal(b []byte) {
	pos := 0
	a.To = string(b[pos:idLength])
	pos += idLength

	a.From = string(b[pos : pos+idLength])
	pos += idLength

	a.RemoteSD = b[pos:]

	return
}

func (c connectionSign) marshal() ([]byte, error) {
	// Проверяем длины Receiver и Sender
	if len(c.To) > idLength {
		return nil, fmt.Errorf("receiver too long: %d bytes (max %d)", len(c.To), idLength)
	}
	if len(c.From) > idLength {
		return nil, fmt.Errorf("sender too long: %d bytes (max %d)", len(c.From), idLength)
	}

	pubKeyBytes, err := crypt.MarshalPublicKey(c.PublicKey)
	if err != nil {
		return nil, err
	}

	keySize := len(pubKeyBytes)
	if keySize > maxPubKeyLength || keySize < minPubKeyLength {
		return nil, fmt.Errorf("invalid public key size: %d bytes (must be between %d and %d)",
			keySize, minPubKeyLength, maxPubKeyLength)
	}

	stunAddrLen := len(c.StunServer)
	if stunAddrLen > maxStunServerLength {
		return nil, fmt.Errorf("STUN address too long: %d bytes (max 128)", stunAddrLen)
	}

	totalSize := idLength + idLength + stunAddrLen + keySize + signLength + 3
	result := make([]byte, totalSize)
	pos := 0

	copy(result[pos:pos+idLength], []byte(c.To))
	pos += idLength

	copy(result[pos:pos+idLength], []byte(c.From))
	pos += idLength

	result[pos] = byte(stunAddrLen)
	pos++

	copy(result[pos:], []byte(c.StunServer))
	pos += stunAddrLen

	binary.BigEndian.PutUint16(result[pos:], uint16(keySize))
	pos += 2

	copy(result[pos:pos+keySize], pubKeyBytes)
	pos += keySize

	copy(result[pos:], []byte(c.Sign))
	return result, nil
}

func (c *connectionSign) unmarshal(data []byte) error {
	pos := 0

	c.To = string(data[pos : pos+idLength])
	pos += idLength

	c.From = string(data[pos : pos+idLength])
	pos += idLength

	stunAddrLen := int(data[pos])
	pos++
	c.Sign = string(data[pos : pos+stunAddrLen])
	pos += stunAddrLen

	keySize := int(binary.BigEndian.Uint16(data[pos:]))
	pos += 2
	if keySize < minPubKeyLength || keySize > maxPubKeyLength {
		return fmt.Errorf("invalid public key size: %d bytes", keySize)
	}
	pubKeyData := data[pos : pos+keySize]
	pubKey, err := crypt.ParsePublicKey(pubKeyData)
	if err != nil {
		return err
	}
	c.PublicKey = pubKey

	if len(data[pos:]) != 256 {
		return fmt.Errorf("invalid sign length: %d bytes", len(data[pos:]))
	}
	c.Sign = string(data[pos:])

	return nil
}

func (st signalType) String() string {
	switch st {
	case DoVerifySignal:
		return "DoVerifySignal"
	case ProvidePubKeySignal:
		return "ProvidePubKeySignal"
	case PubKeyProvidedSignal:
		return "PubKeyProvidedSignal"
	case SolveChallengeSignal:
		return "SolveChallengeSignal"
	case TestChallengeSignal:
		return "TestChallengeSignal"
	case NewConnectionSignal:
		return "NewConnectionSignal"
	case GenerateConnectionSignSignal:
		return "GenerateConnectionSignSignal"
	case SendConnectionSignSignal:
		return "SendConnectionSignSignal"
	case MakeOfferSignal:
		return "MakeOfferSignal"
	case SendOfferSignal:
		return "SendOfferSignal"
	case HandleOfferSignal:
		return "HandleOfferSignal"
	case SendAnswerSignal:
		return "SendAnswerSignal"
	case HandleAnswerSignal:
		return "HandleAnswerSignal"
	case ConnectionEstablishedSignal:
		return "ConnectionEstablishedSignal"
	case PingSignal:
		return "PingSignal"
	case PongSignal:
		return "PongSignal"
	case DisconnectCandidate:
		return "DisconnectCandidate"
	default:
		return "unknown"
	}
}
