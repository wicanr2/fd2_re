package fdother

import "testing"

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
