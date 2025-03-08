package network

import "time"

var (
	minNetworkConns = 5

	waitOfferTimeout = 30 * time.Second

	waitingSignTimeout = 30 * time.Second

	waitRTCAnswer = 30 * time.Second

	waitingConnectionEstablishingTimeout = 30 * time.Second

	maxMessagesPerMinute = 600

	idLength = 52

	maxStunServerLength = 128

	signLength = 52

	pubKeyLength = 512
)
