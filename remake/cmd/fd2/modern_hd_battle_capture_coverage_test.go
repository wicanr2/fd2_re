package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestModernHDPhysicalCaptureZeroDelayContract(t *testing.T) {
	animations := func() map[int]*figani.Animation {
		return map[int]*figani.Animation{
			336: {Frames: []figani.Frame{{Delay: 2}}},
			337: {Frames: []figani.Frame{{Delay: 2}, {Delay: 0}, {Delay: 2}}},
		}
	}
	shots := func() map[string]any {
		return map[string]any{"336-00": true, "337-00": true, "337-02": true}
	}
	if zero, err := checkModernHDPhysicalCaptures(112, animations(), shots(), nil); err != nil || len(zero) != 1 || zero[0] != (modernHDZeroDelaySource{Resource: 337, Frame: 1}) {
		t.Fatalf("zero-delay coverage: %+v %v", zero, err)
	}
	t.Run("missing-visible", func(t *testing.T) {
		values := shots()
		delete(values, "337-02")
		if _, err := checkModernHDPhysicalCaptures(112, animations(), values, nil); err == nil {
			t.Fatal("missing nonzero frame accepted")
		}
	})
	t.Run("zero-captured", func(t *testing.T) {
		values := shots()
		values["337-01"] = true
		if _, err := checkModernHDPhysicalCaptures(112, animations(), values, nil); err == nil {
			t.Fatal("zero-delay capture accepted")
		}
	})
	t.Run("zero-bound", func(t *testing.T) {
		if _, err := checkModernHDPhysicalCaptures(112, animations(), shots(), map[string]bool{"337-01": true}); err == nil {
			t.Fatal("zero-delay layer binding accepted")
		}
	})
	t.Run("idle-zero-still-visible", func(t *testing.T) {
		values, captured := animations(), shots()
		values[336].Frames[0].Delay = 0
		delete(captured, "336-00")
		if _, err := checkModernHDPhysicalCaptures(112, values, captured, nil); err == nil {
			t.Fatal("idle zero was treated as an invisible attack")
		}
	})
	t.Run("departure-zero-still-visible", func(t *testing.T) {
		values := animations()
		values[337].HeaderByte1 = 1
		if _, err := checkModernHDPhysicalCaptures(112, values, shots(), nil); err == nil {
			t.Fatal("departure zero was treated as an invisible tail")
		}
	})
}
