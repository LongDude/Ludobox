package service

import (
	"context"
	"testing"
	"time"
)

func TestUserBalanceEventServicePublishesOnlyToMatchingUser(t *testing.T) {
	svc := NewUserBalanceEventService(nil, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	userEvents, userUnsubscribe := svc.Subscribe(ctx, 10)
	defer userUnsubscribe()
	otherEvents, otherUnsubscribe := svc.Subscribe(ctx, 20)
	defer otherUnsubscribe()

	want := UserBalanceEvent{
		Type:      "user_balance_changed",
		Action:    "update",
		UserID:    10,
		Balance:   250,
		Timestamp: time.Now().UTC(),
	}
	svc.Publish(want)

	select {
	case got := <-userEvents:
		if got.Type != want.Type || got.UserID != want.UserID || got.Balance != want.Balance {
			t.Fatalf("unexpected balance event: got %+v want %+v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for matching user balance event")
	}

	select {
	case got := <-otherEvents:
		t.Fatalf("received balance event for another user: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestUserBalanceEventServiceUnsubscribeRemovesSubscriber(t *testing.T) {
	svc := NewUserBalanceEventService(nil, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, unsubscribe := svc.Subscribe(ctx, 10)
	unsubscribe()

	svc.Publish(UserBalanceEvent{
		Type:      "user_balance_changed",
		Action:    "update",
		UserID:    10,
		Balance:   300,
		Timestamp: time.Now().UTC(),
	})

	select {
	case got := <-events:
		t.Fatalf("received event after unsubscribe: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
}
