package stream

import (
	"testing"
	"time"

	"llm-bb/internal/model"
)

func TestHubCloseClosesSubscribers(t *testing.T) {
	hub := NewHub()
	streamCh, cancel := hub.Subscribe(42)
	defer cancel()

	hub.Close()

	select {
	case _, ok := <-streamCh:
		if ok {
			t.Fatal("expected closed subscriber channel")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscriber channel to close")
	}

	hub.Publish(model.Message{RoomID: 42})

	closedCh, closedCancel := hub.Subscribe(42)
	defer closedCancel()

	select {
	case _, ok := <-closedCh:
		if ok {
			t.Fatal("expected closed channel from late subscription")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for closed late subscription")
	}
}
