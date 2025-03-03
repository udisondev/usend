package model

import "math/big"

type SignalType uint8

const (
	DoVerifySignal               SignalType = 0x00
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

type NetworkSignal struct {
	Type    SignalType
	Payload []byte
}

type IncomeSignal struct {
	From string
	NetworkSignal
}

type Signature struct {
	R, S *big.Int
}

func (st SignalType) String() string {
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
