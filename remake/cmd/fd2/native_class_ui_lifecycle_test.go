package main

import (
	"testing"
	"time"
)

func TestNativeClassUILifecycleRequiresDrawAcknowledgment(t *testing.T) {
	g := &Game{nativeClassUIJob: &nativeClassUIJob{
		frames: [][]byte{{0}, {1}, {2}, {3}, {4}, {5}},
	}}
	g.stepNativeClassUILifecycle(time.Time{})
	g.stepNativeClassUILifecycle(time.Time{})
	if g.nativeClassUIJob.frame != 0 {
		t.Fatalf("undrawn opening frame advanced to %d", g.nativeClassUIJob.frame)
	}
	for want := 0; want < 6; want++ {
		job := g.nativeClassUIJob
		if job == nil || job.frame != want {
			t.Fatalf("present %d has job %#v", want, job)
		}
		job.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeClassUIJob != nil {
		t.Fatal("six-frame list opening did not settle")
	}
}

func TestNativeClassUIClosingDefersContinuationUntilFourthPresent(t *testing.T) {
	completed := false
	g := &Game{nativeClassUIJob: &nativeClassUIJob{
		frames: [][]byte{{0}, {1}, {2}, {3}},
		after:  func() { completed = true },
	}}
	for want := 0; want < 4; want++ {
		if completed || g.nativeClassUIJob.frame != want {
			t.Fatalf("before present %d: completed=%v job=%#v", want, completed, g.nativeClassUIJob)
		}
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if !completed || g.nativeClassUIJob != nil {
		t.Fatalf("closing completion=%v job=%#v", completed, g.nativeClassUIJob)
	}
}

func TestNativeClassUIClosingPresentsRestoreBeforeContinuation(t *testing.T) {
	completed := false
	g := &Game{nativeClassUIJob: &nativeClassUIJob{
		frames:  [][]byte{{0}, {1}},
		restore: make([]byte, 320*200),
		after:   func() { completed = true },
	}}
	for want := 0; want < 2; want++ {
		if completed || g.nativeClassUIJob.frame != want {
			t.Fatalf("before frame %d: completed=%v job=%#v", want, completed, g.nativeClassUIJob)
		}
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if completed || g.nativeClassUIJob == nil || g.nativeClassUIJob.frame != 2 {
		t.Fatalf("continuation ran before restore: completed=%v job=%#v", completed, g.nativeClassUIJob)
	}
	g.nativeClassUIJob.drawn = true
	g.stepNativeClassUILifecycle(time.Time{})
	if !completed || g.nativeClassUIJob != nil {
		t.Fatalf("restore completion=%v job=%#v", completed, g.nativeClassUIJob)
	}
}

func TestNativeClassUIPulseUsesTwoBIOSTickCadenceAndWrap(t *testing.T) {
	g := &Game{}
	check := func(tick, want int) {
		t.Helper()
		g.stepNativeClassUIPulseTick(tick)
		if g.nativeClassUIPulse != want {
			t.Fatalf("tick %#x pulse=%d want %d", tick, g.nativeClassUIPulse, want)
		}
	}
	check(0x7ffd, 0)
	check(0x7ffe, 0)
	check(0x7fff, 1)
	check(-0x8000, 2)
	check(-0x7fff, 2)
	check(-0x7ffe, 3)
	check(-0x7ffc, 0)
}

func TestNativeClassUIPulseMatchesSignedBIOSBranch(t *testing.T) {
	for _, test := range []struct {
		name                           string
		previous, current, pulse, want int
	}{
		{"same", 9, 9, 2, 2},
		{"one", 9, 10, 2, 2},
		{"two", 9, 11, 2, 3},
		{"large", 9, 100, 2, 3},
		{"backward", 9, 8, 2, 3},
		{"signed-crossing", 0x7fff, 0x8000, 3, 0},
		{"unsigned-wrap-one", -1, 0, 3, 3},
		{"unsigned-wrap-two", -1, 1, 3, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := &Game{nativeClassUIPulse: test.pulse, nativeClassUILastTick: test.previous, nativeClassUIHasTick: true}
			g.stepNativeClassUIPulseTick(test.current)
			if g.nativeClassUIPulse != test.want {
				t.Fatalf("pulse=%d want %d", g.nativeClassUIPulse, test.want)
			}
			wantTick := test.previous
			if test.want != test.pulse {
				wantTick = int(int16(uint16(test.current)))
			}
			if g.nativeClassUILastTick != wantTick {
				t.Fatalf("latch=%d want %d", g.nativeClassUILastTick, wantTick)
			}
		})
	}
}

func TestNativeClassUIPulseUsesSecondBIOSReadForLatch(t *testing.T) {
	g := &Game{nativeClassUIHasTick: true, nativeClassUILastTick: 7}
	g.stepNativeClassUIPulseTicks(9, 10)
	if g.nativeClassUIPulse != 1 || g.nativeClassUILastTick != 10 {
		t.Fatal("second read was not saved")
	}
	g.stepNativeClassUIPulseTicks(11, 12)
	if g.nativeClassUIPulse != 1 || g.nativeClassUILastTick != 10 {
		t.Fatal("waiting branch published a new latch")
	}
	g.stepNativeClassUIPulseTicks(12, 13)
	if g.nativeClassUIPulse != 2 || g.nativeClassUILastTick != 13 {
		t.Fatal("second update did not preserve separate samples")
	}
}

func TestNativeClassUITimelineRequiresFinalFramePresentation(t *testing.T) {
	completed := false
	start := time.Unix(100, 0)
	g := &Game{nativeClassUIJob: &nativeClassUIJob{
		timeline: []nativeClassUITimelineStep{
			{frame: []byte{1}, duration: 10 * time.Millisecond},
			{frame: []byte{2}},
		},
		after: func() { completed = true },
	}}
	g.stepNativeClassUILifecycle(start)
	g.nativeClassUIJob.drawn = true
	g.stepNativeClassUILifecycle(start.Add(20 * time.Millisecond))
	if completed || g.nativeClassUIJob == nil {
		t.Fatal("timeline completed before its zero-duration final frame was presented")
	}
	g.nativeClassUIJob.frame = 1
	g.nativeClassUIJob.drawn = true
	g.stepNativeClassUILifecycle(start.Add(20 * time.Millisecond))
	if !completed || g.nativeClassUIJob != nil {
		t.Fatalf("timeline completion=%v job=%#v", completed, g.nativeClassUIJob)
	}
}
