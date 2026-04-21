package service

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestAdminEventServicePublishesToSubscriber(t *testing.T) {
	svc := NewAdminEventService(nil, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, unsubscribe := svc.Subscribe(ctx)
	defer unsubscribe()

	want := AdminEvent{
		Type:      "admin_resource_changed",
		Resource:  "rooms",
		Action:    "update",
		ID:        42,
		Timestamp: time.Now().UTC(),
	}
	svc.Publish(want)

	select {
	case got := <-events:
		if got.Type != want.Type || got.Resource != want.Resource || got.Action != want.Action || got.ID != want.ID {
			t.Fatalf("unexpected event: got %+v want %+v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for admin event")
	}
}

func TestAdminEventServiceUnsubscribeRemovesSubscriber(t *testing.T) {
	svc := NewAdminEventService(nil, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, unsubscribe := svc.Subscribe(ctx)
	unsubscribe()

	svc.Publish(AdminEvent{
		Type:      "admin_resource_changed",
		Resource:  "servers",
		Action:    "update",
		ID:        7,
		Timestamp: time.Now().UTC(),
	})

	select {
	case got := <-events:
		t.Fatalf("received event after unsubscribe: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func testLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}
