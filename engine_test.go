package main

import (
	"reflect"
	"testing"
)

func assertInput(t *testing.T, e *engine, source string, down, consumed bool, want ...output) {
	t.Helper()
	gotConsumed, got := e.process(source, down)
	if gotConsumed != consumed || !reflect.DeepEqual(got, want) {
		t.Fatalf("process(%q, %v) = (%v, %#v), want (%v, %#v)", source, down, gotConsumed, got, consumed, want)
	}
}

func TestSideButtonHoldsMiddleAndReleases(t *testing.T) {
	e := newEngine()
	e.configure(true, []rule{{"mouse_x1", "mouse_middle", true}})
	assertInput(t, e, "mouse_x1", true, true, output{"mouse_middle", true})
	assertInput(t, e, "mouse_x1", true, true) // Duplicate mouse-down cannot click twice.
	assertInput(t, e, "mouse_x1", false, true, output{"mouse_middle", false})
	assertInput(t, e, "A", true, false)
	if len(e.held) != 0 || len(e.refs) != 0 {
		t.Fatal("held keys remain after release")
	}
}

func TestRightReleaseDoesNotChangeSideButtonHold(t *testing.T) {
	e := newEngine()
	e.configure(true, []rule{{"mouse_x1", "mouse_middle", true}})
	assertInput(t, e, "mouse_right", true, false)
	assertInput(t, e, "mouse_x1", true, true, output{"mouse_middle", true})
	assertInput(t, e, "mouse_right", false, false)
	if e.refs["mouse_middle"] != 1 {
		t.Fatal("right release changed the mapped middle hold")
	}
	assertInput(t, e, "mouse_x1", false, true, output{"mouse_middle", false})
	assertInput(t, e, "mouse_right", true, false)
	assertInput(t, e, "mouse_x1", true, true, output{"mouse_middle", true})
	assertInput(t, e, "mouse_x1", false, true, output{"mouse_middle", false})
}

func TestPauseDuringPressDoesNotLeaveStuckKeys(t *testing.T) {
	e := newEngine()
	rules := []rule{{"mouse_x1", "mouse_middle", true}}
	e.configure(true, rules)
	assertInput(t, e, "mouse_x1", true, true, output{"mouse_middle", true})
	if got := e.configure(false, rules); !reflect.DeepEqual(got, []output{{"mouse_middle", false}}) {
		t.Fatalf("pause must release middle, got %v", got)
	}
	assertInput(t, e, "mouse_x1", true, true)
	assertInput(t, e, "mouse_x1", false, true)
	assertInput(t, e, "mouse_x1", true, false)
	e.configure(true, rules)
	assertInput(t, e, "mouse_x1", false, false) // Up without a swallowed down must pass through.
}

func TestEditingHeldRuleDoesNotReleaseWrongTarget(t *testing.T) {
	e := newEngine()
	e.configure(true, []rule{{"A", "B", true}})
	assertInput(t, e, "A", true, true, output{"B", true})
	if got := e.configure(true, []rule{{"A", "C", true}}); !reflect.DeepEqual(got, []output{{"B", false}}) {
		t.Fatalf("editing must release previous target, got %v", got)
	}
	assertInput(t, e, "A", false, true)
	assertInput(t, e, "A", true, true, output{"C", true})
	assertInput(t, e, "A", false, true, output{"C", false})
}

func TestTwoSourcesShareTargetUntilBothRelease(t *testing.T) {
	e := newEngine()
	e.configure(true, []rule{{"mouse_x1", "mouse_middle", true}, {"mouse_x2", "mouse_middle", true}})
	assertInput(t, e, "mouse_x1", true, true, output{"mouse_middle", true})
	assertInput(t, e, "mouse_x2", true, true)
	assertInput(t, e, "mouse_x1", false, true)
	assertInput(t, e, "mouse_x2", false, true, output{"mouse_middle", false})
}

func TestKeyboardRepeatAndDisabledRules(t *testing.T) {
	e := newEngine()
	e.configure(true, []rule{{"A", "B", true}, {"C", "D", false}})
	assertInput(t, e, "A", true, true, output{"B", true})
	assertInput(t, e, "A", true, true, output{"B", true})
	assertInput(t, e, "A", false, true, output{"B", false})
	assertInput(t, e, "C", true, false)
	assertInput(t, e, "C", false, false)
}
