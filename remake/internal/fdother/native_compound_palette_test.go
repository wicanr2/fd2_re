package fdother

import "testing"

func TestInterpolateNativeCompoundDACPreservesRawFormulaAndHalfOpenRange(t *testing.T) {
	baseline := make([]byte, 256*3)
	baseline[0], baseline[1], baseline[2] = 3, 33, 63
	baseline[3], baseline[4], baseline[5] = 63, 3, 33
	baseline[255*3] = 17
	raw := [3]byte{63, 63, 63}
	zero, err := InterpolateNativeCompoundDAC(baseline, 0, 255, 0, raw)
	if err != nil {
		t.Fatal(err)
	}
	full, err := InterpolateNativeCompoundDAC(baseline, 0, 255, 40, raw)
	if err != nil {
		t.Fatal(err)
	}
	if zero[0] != 63 || zero[1] != 63 || zero[2] != 63 || full[0] != 3 || full[1] != 33 || full[2] != 63 ||
		zero[255*3] != 17 || full[255*3] != 17 {
		t.Fatalf("compound DAC zero=%v full=%v index255=%d/%d", zero[:6], full[:6], zero[255*3], full[255*3])
	}
	mid, err := InterpolateNativeCompoundDAC(baseline, 1, 2, 20, raw)
	if err != nil {
		t.Fatal(err)
	}
	if mid[0] != 3 || mid[3] != 63 || mid[4] != 33 || mid[5] != 48 {
		t.Fatalf("compound DAC signed interpolation=%v", mid[:6])
	}
}

func TestInterpolateNativeCompoundDACFailsClosed(t *testing.T) {
	baseline := make([]byte, 256*3)
	for _, input := range []struct{ start, end, delta int }{{-1, 255, 1}, {0, 257, 1}, {2, 2, 1}, {0, 255, 41}} {
		if _, err := InterpolateNativeCompoundDAC(baseline, input.start, input.end, input.delta, [3]byte{}); err == nil {
			t.Fatalf("accepted invalid interpolation %+v", input)
		}
	}
}
