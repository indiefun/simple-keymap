package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	workerProtocol   = 1
	inputRejected    = 1
	inputTimedOut    = 2
	inputServiceLost = 3
)

type workerReply struct {
	Protocol int  `json:"protocol,omitempty"`
	Ready    bool `json:"ready,omitempty"`
	OK       bool `json:"ok,omitempty"`
}

type workerResult struct {
	reply workerReply
	err   error
}

// Keep a stuck Windows input syscall out of the configuration process. There
// is one helper per launch; a timeout disables it, without retries or spawning
// more helpers behind a kernel call that Windows cannot currently cancel.
type inputTransport struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	encoder   *json.Encoder
	requests  chan []output
	replies   chan workerResult
	finished  chan struct{}
	stopped   chan struct{}
	closeOnce sync.Once
	job       *workerJob
	timeout   time.Duration
	failure   int
}

func newInputTransport(cmd *exec.Cmd, timeout time.Duration) (*inputTransport, error) {
	childIn, stdin, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stdout, childOut, err := os.Pipe()
	if err != nil {
		childIn.Close()
		stdin.Close()
		return nil, err
	}
	// Own these pipe endpoints ourselves. Cmd.Wait must not close stdout
	// before the reader has consumed a helper's final acknowledgement.
	cmd.Stdin, cmd.Stdout = childIn, childOut
	t := &inputTransport{
		cmd: cmd, stdin: stdin, stdout: stdout, encoder: json.NewEncoder(stdin),
		requests: make(chan []output),
		replies:  make(chan workerResult, 1), finished: make(chan struct{}), stopped: make(chan struct{}), timeout: timeout,
	}
	t.job, err = newWorkerJob()
	if err != nil {
		childIn.Close()
		childOut.Close()
		stdin.Close()
		stdout.Close()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		t.job.close()
		childIn.Close()
		childOut.Close()
		stdin.Close()
		stdout.Close()
		return nil, err
	}
	childIn.Close()
	childOut.Close()
	go func() { _ = cmd.Wait(); close(t.finished) }()
	if err = t.job.assign(cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		t.close()
		return nil, err
	}
	go func() {
		decoder := json.NewDecoder(stdout)
		var ready workerResult
		ready.err = decoder.Decode(&ready.reply)
		select {
		case t.replies <- ready:
		case <-t.stopped:
			return
		}
		if ready.err != nil {
			return
		}
		for {
			var events []output
			select {
			case events = <-t.requests:
			case <-t.stopped:
				return
			}
			var result workerResult
			result.err = t.encoder.Encode(events)
			if result.err == nil {
				result.err = decoder.Decode(&result.reply)
			}
			select {
			case t.replies <- result:
			case <-t.stopped:
				return
			}
			if result.err != nil {
				return
			}
		}
	}()
	select {
	case result := <-t.replies:
		if result.err == nil && result.reply.Ready && result.reply.Protocol == workerProtocol {
			return t, nil
		}
		if result.err != nil {
			err = fmt.Errorf("输入服务启动失败：%w", result.err)
		} else {
			err = errors.New("输入服务版本不匹配，请重新编译并保留同次生成的两个 exe 文件")
		}
	case <-time.After(3 * time.Second):
		err = errors.New("输入服务启动超时")
	}
	t.close()
	return nil, err
}

func (t *inputTransport) deliver(out []output) bool {
	if t.failure != 0 {
		return false
	}
	for len(out) != 0 {
		count := min(len(out), 64)
		timer := time.NewTimer(t.timeout)
		// The deadline covers both pipe writes and the native acknowledgement.
		select {
		case t.requests <- out[:count]:
		case <-t.stopped:
			t.failure = inputServiceLost
			timer.Stop()
			return false
		case <-timer.C:
			t.failure = inputTimedOut
			t.job.terminate()
			return false
		}
		select {
		case result := <-t.replies:
			if result.err != nil {
				t.failure = inputServiceLost
			} else if !result.reply.OK {
				t.failure = inputRejected
			}
		case <-t.stopped:
			t.failure = inputServiceLost
		case <-timer.C:
			t.failure = inputTimedOut
			// Termination is best effort: a syscall stuck in the Windows
			// kernel may retain its helper thread until Windows restarts.
			t.job.terminate()
		}
		timer.Stop()
		if t.failure != 0 {
			return false
		}
		out = out[count:]
	}
	return true
}

func (t *inputTransport) close() {
	t.closeOnce.Do(func() {
		close(t.stopped)
		// Closing a pipe can itself wait for an outstanding Go file operation.
		// Never let that extend the main process's shutdown deadline.
		go func() { _ = t.stdin.Close(); _ = t.stdout.Close() }()
		select {
		case <-t.finished:
		case <-time.After(250 * time.Millisecond):
			t.job.terminate()
		}
		t.job.close()
	})
}

// Each physical edge remains an individual output. Never replace a held key
// with a synthetic click, delay its release, or depend on another mouse button.
func runInputWorker(reader io.Reader, writer io.Writer, deliver func([]output) bool) error {
	decoder, encoder := json.NewDecoder(reader), json.NewEncoder(writer)
	if err := encoder.Encode(workerReply{Ready: true, Protocol: workerProtocol}); err != nil {
		return err
	}
	pressed := map[string]bool{}
	defer func() {
		var releases []output
		for id := range pressed {
			releases = append(releases, output{Key: id})
		}
		if len(releases) != 0 {
			deliver(releases)
		}
	}()
	for {
		var events []output
		if err := decoder.Decode(&events); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if len(events) == 0 || len(events) > 64 {
			return errors.New("invalid input packet")
		}
		for _, o := range events {
			if _, ok := findKey(o.Key); !ok {
				return errors.New("unknown output key")
			}
		}
		ok := deliver(events)
		for _, o := range events {
			if o.Down {
				pressed[o.Key] = true
			} else if ok {
				delete(pressed, o.Key)
			}
		}
		if err := encoder.Encode(workerReply{OK: ok}); err != nil {
			return err
		}
		if !ok {
			return errors.New("Windows rejected input")
		}
	}
}
