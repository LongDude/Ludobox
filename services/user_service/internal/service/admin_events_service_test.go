package service

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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

func TestParseAdminEventTextNotification(t *testing.T) {
	got, err := parseAdminEventNotification(&pgconn.Notification{
		Payload: "rooms|update|42|2026-04-22T10:15:30.123Z",
	})
	if err != nil {
		t.Fatalf("parse admin event notification: %v", err)
	}

	if got.Type != "admin_resource_changed" {
		t.Fatalf("unexpected type: %s", got.Type)
	}
	if got.Resource != "rooms" || got.Action != "update" || got.ID != 42 {
		t.Fatalf("unexpected event: %+v", got)
	}
	if got.Timestamp.IsZero() {
		t.Fatal("expected timestamp")
	}
	if len(got.Data) != 0 {
		t.Fatalf("database notification should not carry row data: %s", string(got.Data))
	}
}

func TestParseAdminEventJSONNotificationForCompatibility(t *testing.T) {
	got, err := parseAdminEventNotification(&pgconn.Notification{
		Payload: `{"type":"admin_resource_changed","resource":"servers","action":"update","id":7,"timestamp":"2026-04-22T10:15:30Z"}`,
	})
	if err != nil {
		t.Fatalf("parse admin event json notification: %v", err)
	}

	if got.Resource != "servers" || got.Action != "update" || got.ID != 7 {
		t.Fatalf("unexpected event: %+v", got)
	}
}

func testLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}
