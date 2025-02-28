package model

type SignalType uint8

const (
	DoVerify SignalType = 0x00
)

type NetworkSignal struct {
	Type SignalType
}

func (st SignalType) String() string {
	switch st {
	case DoVerify:
		return "DoVerify"
	default:
		return "unknown"
	}
}
