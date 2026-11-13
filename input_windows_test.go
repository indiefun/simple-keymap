//go:build windows

package main

import (
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"
)

func TestWindowsInputLayout(t *testing.T) {
	want := uintptr(40)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		want = 28
	}
	if got := unsafe.Sizeof(winInput{}); got != want {
		t.Fatalf("INPUT size = %d; want %d", got, want)
	}
	for _, tc := range []struct {
		key         string
		down        bool
		flags, data uint32
	}{
		{"mouse_middle", true, 0x20, 0}, {"mouse_middle", false, 0x40, 0},
		{"mouse_x1", true, 0x80, 1}, {"mouse_x2", false, 0x100, 2},
	} {
		i := inputFor(output{tc.key, tc.down})
		m := (*mouseInput)(unsafe.Pointer(&i.Data[0]))
		if i.Type != 0 || m.Flags != tc.flags || m.Data != tc.data || m.Extra != injectionTag {
			t.Fatalf("incorrect mouse input for %s: %#v", tc.key, m)
		}
	}
}

func TestMouseMessageEdgesMapIndependently(t *testing.T) {
	for _, c := range []struct {
		message uintptr
		data    uint32
		id      string
		down    bool
	}{
		{0x020B, 1 << 16, "mouse_x1", true}, {0x020C, 1 << 16, "mouse_x1", false},
		{0x020B, 2 << 16, "mouse_x2", true}, {0x020C, 2 << 16, "mouse_x2", false},
		{0x0207, 0, "mouse_middle", true}, {0x0208, 0, "mouse_middle", false},
		{0x0204, 0, "mouse_right", true}, {0x0205, 0, "mouse_right", false},
	} {
		id, down := mouseButtonEvent(c.message, c.data)
		if id != c.id || down != c.down {
			t.Fatalf("message %x -> %q %v; want %q %v", c.message, id, down, c.id, c.down)
		}
	}
}

func TestSideButtonClickPipelineProducesTwoMiddleDownUpPairs(t *testing.T) {
	e := newEngine()
	e.configure(true, []rule{{"mouse_x1", "mouse_middle", true}})
	var flags []uint32
	for _, message := range []uintptr{0x020B, 0x020C, 0x0205, 0x0204, 0x020B, 0x020C} {
		id, down := mouseButtonEvent(message, 1<<16)
		consumed, out := e.process(id, down)
		if consumed != (id == "mouse_x1") {
			t.Fatalf("wrong source interception for %x", message)
		}
		for _, o := range out {
			input := inputFor(o)
			mouse := (*mouseInput)(unsafe.Pointer(&input.Data[0]))
			if input.Type != 0 || mouse.Data != 0 {
				t.Fatal("side key generated a non-middle input")
			}
			flags = append(flags, mouse.Flags)
		}
	}
	if len(flags) != 4 || flags[0] != 0x20 || flags[1] != 0x40 || flags[2] != 0x20 || flags[3] != 0x40 {
		t.Fatalf("two side clicks did not become two middle down/up pairs: %x", flags)
	}
	if len(e.held) != 0 || len(e.refs) != 0 {
		t.Fatal("second click left a pressed key")
	}
}

func TestMouseHookReturnsWhileNativeSenderIsBlocked(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	previous := app
	a := &application{engine: newEngine()}
	app = a
	defer func() { app = previous }()
	started, unblock := make(chan struct{}), make(chan struct{})
	var events []output
	a.sender = newInputSender(func(batch []output) bool {
		if len(events) == 0 {
			close(started)
			<-unblock
		}
		events = append(events, batch...)
		return true
	}, nil)
	defer func() { a.sender.close(); awaitSender(t, a.sender.done) }()
	defer close(unblock)
	a.engine.configure(true, []rule{{"mouse_x1", "mouse_middle", true}})
	_, out := a.engine.process("mouse_x1", true)
	a.emit(out)
	awaitSender(t, started)
	// Run the exact mouse hook path from the captured hang. The matching
	// release must be swallowed and queued while SendInput is still waiting.
	released := make(chan uintptr, 1)
	go func() { released <- mouseProc(0, 0x020C, &mouseHookData{Data: 1 << 16}) }()
	select {
	case result := <-released:
		if result != 1 {
			t.Fatalf("mapped mouse-up was not swallowed: %d", result)
		}
	case <-time.After(time.Second):
		t.Fatal("mouse hook is blocked by native input delivery")
	}
	if len(a.engine.held) != 0 {
		t.Fatal("mouse release did not clear held source")
	}
}

// Exercise our own window controller with real native controls, without sending
// input to the user's desktop or depending on a physical mouse with side buttons.
func TestNativeRuleAndCaptureFlow(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	previous := app
	app = &application{
		controls: map[int]uintptr{}, modifiers: map[string]bool{}, dpi: 96,
		path: t.TempDir() + "\\config.json", cfg: config{Enabled: true}, engine: newEngine(),
	}
	a := app
	defer func() { app = previous }()
	if err := a.create(); err != nil {
		t.Fatal(err)
	}
	defer a.dispose()
	defer destroyWindow.Call(a.hwnd)
	a.apply()
	a.refreshList()
	a.command(idAdd) // The initial selections are X1 -> middle.
	if len(a.cfg.Rules) != 1 || a.cfg.Rules[0].Source != "mouse_x1" || a.cfg.Rules[0].Target != "mouse_middle" {
		t.Fatalf("unexpected default rule: %#v", a.cfg.Rules)
	}
	if n := send(a.controls[idList], lvmFirst+4, 0, 0); n != 1 {
		t.Fatalf("native list has %d rows", n)
	}
	if s := send(a.controls[idList], lvmFirst+44, 0, 0xF000); s != 0x2000 {
		t.Fatalf("new rule not checked: %x", s)
	}
	// Changing the native checkbox must update engine, label and persisted config.
	item := listItem{State: 0x1000, StateMask: 0xF000}
	send(a.controls[idList], lvmFirst+43, 0, uintptr(unsafe.Pointer(&item)))
	if a.cfg.Rules[0].Enabled {
		t.Fatal("unchecking did not pause the rule")
	}
	cfg, err := loadConfig(a.path)
	if err != nil || cfg.Rules[0].Enabled {
		t.Fatalf("checkbox not persisted: %#v, %v", cfg, err)
	}
	a.startCapture(idRecordSource)
	if a.engine.enabled {
		t.Fatal("mapping must pause while recording")
	}
	a.finishCapture(6) // A is the first keyboard item after the five mouse buttons.
	if i := send(a.controls[idSource], cbGetCurSel, 0, 0); keys[i].ID != "A" {
		t.Fatalf("recorded selection = %d", i)
	}
	if !a.engine.enabled {
		t.Fatal("mapping was not restored after recording")
	}
	a.command(idToggle)
	if a.cfg.Enabled || a.engine.enabled {
		t.Fatal("global toggle did not pause")
	}
	item = listItem{State: 0x2, StateMask: 0x2}
	send(a.controls[idList], lvmFirst+43, 0, uintptr(unsafe.Pointer(&item)))
	a.command(idDelete)
	if len(a.cfg.Rules) != 0 || send(a.controls[idList], lvmFirst+4, 0, 0) != 0 {
		t.Fatal("delete did not remove native rule")
	}
	// Native messages must hand outputs to the sender only after the hook
	// callback returned, and a timed-out sender must never appear enabled.
	received := make(chan []output, 1)
	a.sender = newInputSender(func(batch []output) bool {
		received <- append([]output(nil), batch...)
		return true
	}, nil)
	defer func() { a.sender.close(); awaitSender(t, a.sender.done) }()
	a.emit([]output{{"mouse_middle", true}, {"mouse_middle", false}})
	if len(a.pendingOutput) != 2 {
		t.Fatal("physical edges were not deferred to the message loop")
	}
	windowProc(a.hwnd, wmSendOutput, 0, 0)
	select {
	case edges := <-received:
		if len(edges) != 2 || !edges[0].Down || edges[1].Down {
			t.Fatalf("incorrect deferred edges: %v", edges)
		}
	case <-time.After(time.Second):
		t.Fatal("message loop did not hand off the edges")
	}
	a.cfg.Enabled = true
	a.apply()
	windowProc(a.hwnd, wmInputError, inputTimedOut, 0)
	if a.cfg.Enabled || a.engine.enabled || !strings.Contains(a.status, "超时") {
		t.Fatalf("timeout did not pause visibly: %q", a.status)
	}
	windowProc(a.hwnd, wmHotkey, 0, 0)
	if a.cfg.Enabled || a.engine.enabled {
		t.Fatal("hotkey re-enabled a failed input service")
	}
}

func TestWindowsKeyboardExtendedAndRelease(t *testing.T) {
	for _, id := range []string{"A", "rctrl", "left", "num_enter", "media_play"} {
		k, _ := findKey(id)
		for _, down := range []bool{true, false} {
			i := inputFor(output{id, down})
			kb := (*keyboardInput)(unsafe.Pointer(&i.Data[0]))
			if i.Type != 1 || (kb.Flags&1 != 0) != k.Extended || (kb.Flags&2 == 0) != down || kb.Extra != injectionTag {
				t.Fatalf("incorrect keyboard input for %s: %#v", id, kb)
			}
		}
	}
	if keyboardKey(0x0D, 0, 1) != "num_enter" || keyboardKey(0x11, 0, 1) != "rctrl" {
		t.Fatal("extended input decoded incorrectly")
	}
	if keyboardKey(0x24, 0x47, 0) != "num7" || keyboardKey(0x24, 0x47, 1) != "home" {
		t.Fatal("Num Lock changed the numeric keypad's physical mapping")
	}
}

func TestEmergencyShortcutClearsPreviouslyMappedF12(t *testing.T) {
	a := &application{engine: newEngine()}
	a.engine.configure(true, []rule{{"F12", "A", true}})
	a.engine.process("F12", true)
	a.releaseReservedF12()
	if _, held := a.engine.held["F12"]; held {
		t.Fatal("reserved shortcut left the old F12 hold behind")
	}
	if len(a.engine.refs) != 0 {
		t.Fatal("reserved shortcut left its mapped target pressed")
	}
	consumed, out := a.engine.process("F12", true)
	if !consumed || len(out) != 1 || !out[0].Down {
		t.Fatal("next F12 press was lost after emergency pause")
	}
}
