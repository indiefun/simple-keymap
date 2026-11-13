package main

import (
	"reflect"
	"testing"
	"time"
)

func awaitSender(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("input sender did not finish")
	}
}

func TestSenderReturnsWhileSendInputIsBlocked(t *testing.T) {
	started, unblock := make(chan struct{}), make(chan struct{})
	var got []output
	s := newInputSender(func(batch []output) bool {
		if len(got) == 0 {
			close(started)
			<-unblock
		}
		got = append(got, batch...)
		return true
	}, nil)
	s.enqueue([]output{{"mouse_middle", true}})
	awaitSender(t, started)
	// This is the former deadlock: the hook must return without waiting for
	// SendInput to finish, and a subsequent physical release must not be lost.
	returned := make(chan struct{})
	go func() {
		s.enqueue([]output{{"mouse_middle", false}})
		s.close()
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(time.Second):
		close(unblock)
		t.Fatal("hook enqueue / close blocked behind SendInput")
	}
	close(unblock)
	awaitSender(t, s.done)
	want := []output{{"mouse_middle", true}, {"mouse_middle", false}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestSenderPreservesOrderUnderRepeatedMapping(t *testing.T) {
	const clicks = 50000
	var got []output
	delivered := make(chan struct{})
	s := newInputSender(func(batch []output) bool {
		got = append(got, batch...)
		if len(got) == clicks*2 {
			close(delivered)
		}
		return true
	}, nil)
	for i := 0; i < clicks; i++ {
		s.enqueue([]output{{"mouse_middle", true}})
		s.enqueue([]output{{"mouse_middle", false}})
	}
	awaitSender(t, delivered)
	s.close()
	awaitSender(t, s.done)
	if len(got) != clicks*2 {
		t.Fatalf("received %d events, want %d", len(got), clicks*2)
	}
	for i, o := range got {
		if o.Key != "mouse_middle" || o.Down != (i%2 == 0) {
			t.Fatalf("event %d out of order: %v", i, o)
		}
	}
}

func TestSenderReleasesHeldOutputOnShutdown(t *testing.T) {
	var got []output
	pressed := make(chan struct{}, 1)
	s := newInputSender(func(batch []output) bool {
		got = append(got, batch...)
		if batch[0].Down {
			pressed <- struct{}{}
		}
		return true
	}, nil)
	s.enqueue([]output{{"mouse_middle", true}})
	select {
	case <-pressed:
	case <-time.After(time.Second):
		t.Fatal("down was not delivered")
	}
	s.close()
	awaitSender(t, s.done)
	want := []output{{"mouse_middle", true}, {"mouse_middle", false}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shutdown events = %v, want %v", got, want)
	}
}

func TestSenderStopsAfterFirstFailure(t *testing.T) {
	var got []output
	failures := 0
	s := newInputSender(func(batch []output) bool {
		got = append(got, batch...)
		return false
	}, func() { failures++ })
	s.enqueue([]output{{"mouse_middle", true}})
	awaitSender(t, s.done)
	s.enqueue([]output{{"mouse_middle", false}})
	s.close()
	if failures != 1 || len(got) != 1 {
		t.Fatalf("failure recovery = %d, %v", failures, got)
	}
}

func TestClosingDiscardsQueuedClicksAndReleasesInFlightHold(t *testing.T) {
	started, unblock := make(chan struct{}), make(chan struct{})
	var got []output
	s := newInputSender(func(batch []output) bool {
		if len(got) == 0 {
			close(started)
			<-unblock
		}
		got = append(got, batch...)
		return true
	}, nil)
	s.enqueue([]output{{"mouse_middle", true}})
	awaitSender(t, started)
	s.enqueue([]output{{"mouse_middle", false}, {"mouse_middle", true}, {"mouse_middle", false}, {"A", true}})
	s.close()
	close(unblock)
	awaitSender(t, s.done)
	want := []output{{"mouse_middle", true}, {"mouse_middle", false}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("close replayed queued clicks: %v", got)
	}
}
