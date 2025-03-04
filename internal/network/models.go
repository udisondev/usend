package network

import (
	"crypto/rsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
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

func (p connectionSign) Marshal() ([]byte, error) {
	// Проверяем длины Receiver и Sender
	if len(p.To) > idLength {
		return nil, fmt.Errorf("receiver too long: %d bytes (max %d)", len(p.To), idLength)
	}
	if len(p.From) > idLength {
		return nil, fmt.Errorf("sender too long: %d bytes (max %d)", len(p.From), idLength)
	}

	pubKeyBytes, err := crypt.MarshalPublicKey(p.PublicKey)
	if err != nil {
		return nil, err
	}

	keySize := len(pubKeyBytes)
	if keySize > maxPubKeyLength || keySize < minPubKeyLength {
		return nil, fmt.Errorf("invalid public key size: %d bytes (must be between %d and %d)",
			keySize, minPubKeyLength, maxPubKeyLength)
	}

	stunAddrLen := len(p.StunServer)
	if stunAddrLen > maxStunServerLength {
		return nil, fmt.Errorf("STUN address too long: %d bytes (max 128)", stunAddrLen)
	}

	totalSize := idLength + idLength + 1 + stunAddrLen + 2 + keySize
	result := make([]byte, totalSize)
	pos := 0

	copy(result[pos:pos+idLength], p.To)
	pos += idLength

	copy(result[pos:pos+idLength], p.From)
	pos += idLength

	result[pos] = byte(stunAddrLen)
	pos++

	copy(result[pos:], p.StunServer)
	pos += stunAddrLen

	binary.BigEndian.PutUint16(result[pos:], uint16(keySize))
	pos += 2

	copy(result[pos:], pubKeyBytes)

	return result, nil
}

func Unmarshal(data []byte) (*connectionSign, error) {
	if len(data) < idLength+idLength+1 {
		return nil, errors.New("data too short")
	}

	p := &connectionSign{}
	pos := 0

	p.To = string(data[pos : pos+idLength])
	pos += idLength

	p.From = string(data[pos : pos+idLength])
	pos += idLength

	stunAddrLen := int(data[pos])
	pos++

	if pos+stunAddrLen > len(data) {
		return nil, errors.New("incomplete packet: invalid STUN address length")
	}
	p.StunServer = string(data[pos : pos+stunAddrLen])
	pos += stunAddrLen

	if pos+2 > len(data) {
		return nil, errors.New("incomplete packet: missing public key length")
	}
	keySize := int(binary.BigEndian.Uint16(data[pos:]))
	pos += 2

	if keySize > maxPubKeyLength || keySize < minPubKeyLength {
		return nil, fmt.Errorf("invalid public key size: %d bytes", keySize)
	}

	if pos+keySize > len(data) {
		return nil, errors.New("incomplete packet: invalid public key data")
	}
	pubKeyData := data[pos : pos+keySize]

	pubKey, err := crypt.ParsePublicKey(pubKeyData)
	if err != nil {
		return nil, err
	}

	p.PublicKey = pubKey

	return p, nil
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
