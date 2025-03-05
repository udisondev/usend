package network

import "time"

var (
	minNetworkConns = 5

	waitOfferTimeout = 30 * time.Second

	waitingSignTimeout = 30 * time.Second

	waitRTCAnswer = 30 * time.Second

	waitingConnectionEstablishingTimeout = 30 * time.Second

	maxMessagesPerMinute = 600

	maxConnSignLength = idLength*2 + maxPubKeyLength + signLength + maxStunServerLength + 3

	minConnSignLength = maxConnSignLength - signLength - (maxPubKeyLength - minPubKeyLength) + 3

	idLength = 52

	maxStunServerLength = 128

	signLength = 256

	maxPubKeyLength = 512

	minPubKeyLength = 256
)
