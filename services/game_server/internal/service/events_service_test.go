package service

import (
	"context"
	"testing"
)

func TestPublishRoundFinalizedSurvivesFullSubscriberBuffer(t *testing.T) {
	events := NewEventsService(nil, nil)
	channel := events.Subscribe(1)
	defer events.Unsubscribe(1, channel)

	for index := 0; index < subscriberBufferSize; index++ {
		events.PublishRoundTimer(context.Background(), 1, "active", subscriberBufferSize-index)
	}

	events.PublishRoundFinalized(context.Background(), 1, nil, nil, 2, 5)

	foundFinalized := false
	drained := 0
	for {
		select {
		case event := <-channel:
			drained++
			if event != nil && event.Type == "round_finalized" {
				foundFinalized = true
			}
		default:
			if !foundFinalized {
				t.Fatalf("expected round_finalized to be delivered, drained=%d", drained)
			}
			return
		}
	}
}
