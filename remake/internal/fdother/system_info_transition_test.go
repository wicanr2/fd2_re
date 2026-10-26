package fdother

import (
	"bytes"
	"testing"
)

func TestNativeSystemInfoRegionsUseAllOriginalCopies(t *testing.T) {
	base, surface := make([]byte, 64000), make([]byte, 64000)
	for i := range base {
		base[i], surface[i] = byte(i%251), byte((i*7+31)%253)
	}
	opening, closing, err := NativeSystemInfoTransitionFrames(base, surface)
	if err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"opening", "closing", "steady"} {
		count := 12
		if phase == "steady" {
			count = 1
		}
		for frame := 0; frame < count; frame++ {
			regions, err := NativeSystemInfoRegions(phase, frame)
			if err != nil {
				t.Fatal(err)
			}
			out := append([]byte(nil), base...)
			for _, region := range regions {
				r := region.Source
				for y := 0; y < r.Dy(); y++ {
					copy(out[(region.Destination.Y+y)*320+region.Destination.X:][:r.Dx()], surface[(r.Min.Y+y)*320+r.Min.X:][:r.Dx()])
				}
			}
			var want []byte
			switch phase {
			case "opening":
				want = opening[frame]
			case "closing":
				want = closing[frame]
			default:
				want, err = NativeSystemInfoSteadyFrame(base, surface)
			}
			if err != nil || !bytes.Equal(out, want) {
				t.Fatal(phase, frame, err)
			}
		}
	}
	for _, phase := range []string{"", "other", "steady", "opening", "closing"} {
		for _, frame := range []int{-1, 12} {
			if _, err := NativeSystemInfoRegions(phase, frame); err == nil {
				t.Fatal("accepted invalid region", phase, frame)
			}
		}
	}
}

func TestNativeSystemInfoSteadyPreservesBattlefieldOutsidePanels(t *testing.T) {
	baseline := make([]byte, NativeSystemInfoBytes)
	information := make([]byte, NativeSystemInfoBytes)
	for i := range baseline {
		baseline[i] = 17
		information[i] = 31
	}
	frame, err := NativeSystemInfoSteadyFrame(baseline, information)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range [][2]int{{0, 0}, {319, 199}, {74, 100}, {245, 100}, {108, 25}} {
		if frame[p[1]*320+p[0]] != 17 {
			t.Fatalf("battlefield changed at %v", p)
		}
	}
	for _, p := range [][2]int{{109, 19}, {75, 37}, {75, 155}, {129, 172}} {
		if frame[p[1]*320+p[0]] != 31 {
			t.Fatalf("panel missing at %v", p)
		}
	}
	if baseline[37*320+75] != 17 {
		t.Fatal("baseline was mutated")
	}
}

func TestNativeSystemInfoTransitionMatches1B1E7TwelvePassOrder(t *testing.T) {
	baseline := make([]byte, NativeSystemInfoBytes)
	information := make([]byte, NativeSystemInfoBytes)
	for i := range information {
		information[i] = 0x7d
	}
	opening, closing, err := NativeSystemInfoTransitionFrames(baseline, information)
	if err != nil {
		t.Fatal(err)
	}
	if len(opening) != 12 || len(closing) != 12 {
		t.Fatalf("transition counts opening=%d closing=%d", len(opening), len(closing))
	}
	pixel := func(frame []byte, x, y int) byte { return frame[y*320+x] }
	if pixel(opening[0], 275, 37) != 0x7d || pixel(opening[0], 274, 37) != 0 ||
		pixel(opening[4], 75, 37) != 0x7d {
		t.Fatalf("horizontal opening boundary does not match 0x1AF99")
	}
	if pixel(opening[2], 109, 43) != 0 || pixel(opening[3], 109, 43) != 0x7d ||
		pixel(opening[7], 109, 19) != 0x7d {
		t.Fatalf("upper opening boundary does not match 0x1B019")
	}
	if pixel(opening[5], 75, 182) != 0 || pixel(opening[6], 75, 182) != 0x7d ||
		pixel(opening[9], 75, 155) != 0x7d {
		t.Fatalf("lower opening boundary does not match 0x1B0AD")
	}
	if pixel(opening[8], 129, 184) != 0 || pixel(opening[9], 129, 184) != 0x7d ||
		pixel(opening[11], 129, 176) != 0x7d || pixel(opening[11], 129, 172) != 0 {
		t.Fatalf("footer opening boundary does not match 0x1B14B")
	}
	if pixel(closing[0], 75, 37) != 0x7d || pixel(closing[7], 75, 37) != 0x7d ||
		pixel(closing[11], 0, 37) != 0x7d || pixel(closing[11], 45, 37) != 0 {
		t.Fatalf("closing pass does not follow 11..0 0x1AF1E order")
	}
}

func TestNativeSystemInfoTransitionRejectsPartialSurfacesAtomically(t *testing.T) {
	if opening, closing, err := NativeSystemInfoTransitionFrames(
		make([]byte, NativeSystemInfoBytes-1), make([]byte, NativeSystemInfoBytes),
	); err == nil || opening != nil || closing != nil {
		t.Fatalf("partial surfaces were accepted: opening=%v closing=%v err=%v", opening, closing, err)
	}
}
