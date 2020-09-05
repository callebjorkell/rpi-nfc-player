//+build !pi

package nfc

import (
	"sync"
	"time"
)

func CreateReader() (CardReader, error) {
	return mockReader{
		init:   &sync.Once{},
		events: make(chan CardEvent, 2),
	}, nil
}

type mockReader struct {
	init   *sync.Once
	events chan CardEvent
}

func (m mockReader) Close() error {
	return nil
}

func (m mockReader) Events() <-chan CardEvent {
	m.init.Do(
		func() {
			go func() {
				for {
					m.events <- CardEvent{
						CardID: "666",
						State:  Activated,
					}
					<-time.After(30 * time.Second)
					m.events <- CardEvent{
						CardID: "666",
						State:  Deactivated,
					}
					<-time.After(10 * time.Second)
				}
			}()
		})

	return m.events
}
