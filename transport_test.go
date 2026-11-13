package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func helperCommand(t *testing.T, mode, capture string) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestInputTransportHelper$")
	cmd.Env = append(os.Environ(), "SIMPLE_KEYMAP_HELPER_MODE="+mode, "SIMPLE_KEYMAP_CAPTURE="+capture)
	return cmd
}

func TestInputTransportHelper(t *testing.T) {
	mode := os.Getenv("SIMPLE_KEYMAP_HELPER_MODE")
	if mode == "" {
		return
	}
	if mode == "no-handshake" {
		time.Sleep(time.Hour)
		return
	}
	if mode == "no-read" {
		_ = json.NewEncoder(os.Stdout).Encode(workerReply{Ready: true, Protocol: workerProtocol})
		time.Sleep(time.Hour)
		return
	}
	var captured []output
	err := runInputWorker(os.Stdin, os.Stdout, func(events []output) bool {
		if mode == "hang" {
			time.Sleep(time.Hour)
		}
		if mode == "reject" {
			return false
		}
		captured = append(captured, events...)
		if path := os.Getenv("SIMPLE_KEYMAP_CAPTURE"); path != "" {
			data, _ := json.Marshal(captured)
			if err := os.WriteFile(path, data, 0600); err != nil {
				return false
			}
		}
		return true
	})
	if path := os.Getenv("SIMPLE_KEYMAP_CAPTURE"); path != "" {
		data, _ := json.Marshal(captured)
		if writeErr := os.WriteFile(path, data, 0600); writeErr != nil {
			os.Exit(2)
		}
	}
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestTransportCloseDoesNotWaitForBlockedAcknowledgement(t *testing.T) {
	transport, err := newInputTransport(helperCommand(t, "hang", ""), 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	delivered := make(chan struct{})
	go func() { transport.deliver([]output{{"mouse_middle", true}}); close(delivered) }()
	// close is valid while a delivery is waiting, including while the helper
	// has not read the packet yet. It must cancel the wait and kill its job.
	started := time.Now()
	transport.close()
	if time.Since(started) > time.Second {
		t.Fatal("closing waited for the 10-second input deadline")
	}
	awaitSender(t, delivered)
	awaitSender(t, transport.finished)
}

func TestTransportDeadlineIncludesUnreadPipe(t *testing.T) {
	transport, err := newInputTransport(helperCommand(t, "no-read", ""), 80*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.close()
	started := time.Now()
	// Larger than the pipe buffer: the deadline must cover Encode as well as
	// waiting for an acknowledgement, even if a worker stops reading entirely.
	if transport.deliver([]output{{Key: string(make([]byte, 128*1024)), Down: true}}) {
		t.Fatal("unread pipe succeeded")
	}
	if transport.failure != inputTimedOut || time.Since(started) > time.Second {
		t.Fatal("pipe write escaped its deadline")
	}
}

func TestTransportTwoClicksWithRightReleaseBetween(t *testing.T) {
	path := filepath.Join(t.TempDir(), "received.json")
	transport, err := newInputTransport(helperCommand(t, "record", path), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.close()
	e := newEngine()
	e.configure(true, []rule{{"mouse_x1", "mouse_middle", true}})
	for _, event := range []output{
		{"mouse_right", true}, {"mouse_x1", true}, {"mouse_x1", false}, {"mouse_right", false},
		{"mouse_right", true}, {"mouse_x1", true}, {"mouse_x1", false}, {"mouse_right", false},
	} {
		consumed, out := e.process(event.Key, event.Down)
		if consumed != (event.Key == "mouse_x1") {
			t.Fatalf("wrong interception for %v", event)
		}
		if !transport.deliver(out) {
			t.Fatalf("input delivery failed: %d", transport.failure)
		}
	}
	transport.close()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got []output
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := []output{{"mouse_middle", true}, {"mouse_middle", false}, {"mouse_middle", true}, {"mouse_middle", false}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("helper received %v, want %v", got, want)
	}
	if len(e.held) != 0 || len(e.refs) != 0 {
		t.Fatal("second click left mapping state held")
	}
}

func TestTransportTimesOutOnceAndRefusesFurtherInput(t *testing.T) {
	transport, err := newInputTransport(helperCommand(t, "hang", ""), 80*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.close()
	started := time.Now()
	if transport.deliver([]output{{"mouse_middle", true}}) || transport.failure != inputTimedOut {
		t.Fatal("blocked native call was not reported as timed out")
	}
	if time.Since(started) > time.Second {
		t.Fatal("timeout did not bound the blocked helper")
	}
	started = time.Now()
	if transport.deliver([]output{{"mouse_middle", false}}) {
		t.Fatal("failed helper accepted another event")
	}
	if time.Since(started) > 50*time.Millisecond {
		t.Fatal("another send piled up after the timeout")
	}
}

func TestTransportReportsRejectedInput(t *testing.T) {
	transport, err := newInputTransport(helperCommand(t, "reject", ""), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.close()
	if transport.deliver([]output{{"mouse_middle", true}}) || transport.failure == 0 {
		t.Fatal("native rejection was reported as successful")
	}
}

func TestWorkerKeepsHoldUntilExplicitRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "received.json")
	transport, err := newInputTransport(helperCommand(t, "record", path), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.close()
	if !transport.deliver([]output{{"mouse_middle", true}}) {
		t.Fatal("down failed")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var held []output
	if err = json.Unmarshal(before, &held); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(held, []output{{"mouse_middle", true}}) {
		t.Fatalf("the first send inserted an unsolicited release: %v", held)
	}
	// EOF invokes the helper's final release only because the source remained
	// held. It must not synthesize that release in the first acknowledged send.
	transport.close()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got []output
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := []output{{"mouse_middle", true}, {"mouse_middle", false}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("hold / exit release = %v, want %v", got, want)
	}
}
