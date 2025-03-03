package network

import "time"

var (
	minNetworkConns                      = 5
	waitingSignTimeout                   = 30 * time.Second
	waitingConnectionEstablishingTimeout = 30 * time.Second
	maxMessagesPerMinute                 = 600
)
