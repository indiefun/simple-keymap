//go:build windows

package main

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func lifecycleCommand(t *testing.T, mode string) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestApplicationLifecycleHelper$")
	cmd.Env = append(os.Environ(), "SIMPLE_KEYMAP_LIFECYCLE="+mode)
	return cmd
}

func TestApplicationLifecycleHelper(t *testing.T) {
	mode := os.Getenv("SIMPLE_KEYMAP_LIFECYCLE")
	if mode == "" {
		return
	}
	if mode == "owner" {
		transport, err := newInputTransport(helperCommand(t, "no-read", ""), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer transport.close()
		_ = json.NewEncoder(os.Stdout).Encode(transport.cmd.Process.Pid)
		time.Sleep(time.Hour)
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	app = &application{
		controls: map[int]uintptr{}, modifiers: map[string]bool{}, dpi: 96,
		path: filepath.Join(t.TempDir(), "config.json"), cfg: config{Enabled: true}, engine: newEngine(),
	}
	a := app
	if err := a.create(); err != nil {
		t.Fatal(err)
	}
	defer a.dispose()
	backend := "record"
	if mode == "close-hang" || mode == "timeout-close" {
		backend = "hang"
	}
	transport, err := newInputTransport(helperCommand(t, backend, ""), 80*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	a.sender = newInputSender(transport.deliver, func() {
		postMessage.Call(a.hwnd, wmInputError, uintptr(transport.failure), 0)
	})
	// The exact production message loop and close path are used here. Native
	// input and global hooks are replaced, so this cannot affect the desktop.
	a.emit([]output{{"mouse_middle", true}})
	go func() {
		if mode == "timeout-close" {
			time.Sleep(200 * time.Millisecond)
		} else {
			time.Sleep(25 * time.Millisecond)
		}
		postMessage.Call(a.hwnd, wmClose, 0, 0)
	}()
	started := time.Now()
	runMessageLoop(a.hwnd)
	a.shutdownInput(transport)
	if time.Since(started) > 1500*time.Millisecond {
		t.Fatal("window close exceeded its deadline")
	}
	if !a.closing {
		t.Fatal("close did not stop mapping")
	}
	awaitSender(t, transport.finished)
}

func TestWindowClosesWithHealthyOrUnresponsiveInputService(t *testing.T) {
	for _, mode := range []string{"normal-close", "close-hang", "timeout-close"} {
		t.Run(mode, func(t *testing.T) {
			cmd := lifecycleCommand(t, mode)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(3 * time.Second):
				_ = cmd.Process.Kill()
				t.Fatal("application subprocess could not exit")
			}
		})
	}
}

func TestInputServiceDiesWhenOwnerIsTerminated(t *testing.T) {
	owner := lifecycleCommand(t, "owner")
	stdout, err := owner.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = owner.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = owner.Process.Kill(); _ = owner.Wait() }()
	pid := make(chan int, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			var id int
			_ = json.Unmarshal(scanner.Bytes(), &id)
			pid <- id
		}
	}()
	var workerPID int
	select {
	case workerPID = <-pid:
	case <-time.After(3 * time.Second):
		t.Fatal("owner did not report helper")
	}
	if workerPID <= 0 {
		t.Fatal("invalid helper pid")
	}
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(workerPID))
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(handle)
	if err = owner.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	state, err := syscall.WaitForSingleObject(handle, 1500)
	if err != nil || state != syscall.WAIT_OBJECT_0 {
		t.Fatalf("input helper survived owner termination: state %d, %v", state, err)
	}
}

func TestMissingSidecarCannotFallBackToMainExecutable(t *testing.T) {
	cmd, err := inputWorkerCommand(filepath.Join(t.TempDir(), "SimpleKeymap.exe"))
	if err == nil || cmd != nil {
		t.Fatal("missing sidecar can still launch the main exe as an input worker")
	}
}
