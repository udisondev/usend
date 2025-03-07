package network

import "time"

func (i *interaction) muteNotVerifiedFilter(in <-chan incomeSignal) <-chan incomeSignal {
	out := make(chan incomeSignal)

	go func() {
		defer close(out)
		for msg := range in {
			i.mu.RLock()
			state := i.state
			i.mu.RUnlock()

			if state == NotVerified {
				return
			}
			out <- msg
		}

	}()

	return out

}

func (i *interaction) messagesPerMinuteFilter(in <-chan incomeSignal) <-chan incomeSignal {
	out := make(chan incomeSignal)

	go func() {
		defer close(out)

		counter := 0
		for {
			select {
			case <-time.After(time.Minute):
				counter = 0
			case msg, ok := <-in:
				if !ok {
					return
				}

				if counter > maxMessagesPerMinute {
					return
				}

				counter++

				out <- msg
			}
		}
	}()

	return out
}
