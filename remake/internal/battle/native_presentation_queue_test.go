package battle

import "testing"

func TestAppendNativePresentationMissUsesOriginalHexTable(t *testing.T) {
	first := NativePresentationDigit{PositionCode: 7, Target: 1, Digit: 94}
	queue := AppendNativePresentationMiss([]NativePresentationDigit{first}, 9)
	want := []NativePresentationDigit{first,
		{PositionCode: 2, Target: 9, Digit: 116},
		{PositionCode: 8, Target: 9, Digit: 117},
		{PositionCode: 12, Target: 9, Digit: 118},
		{PositionCode: 17, Target: 9, Digit: 118},
	}
	if len(queue) != len(want) {
		t.Fatalf("queue=%v", queue)
	}
	for i := range want {
		if queue[i] != want[i] {
			t.Fatalf("entry %d=%+v, want %+v", i, queue[i], want[i])
		}
	}
}

func TestAppendNativePresentationDigitsRightAlignsRawDigits(t *testing.T) {
	queue, err := AppendNativePresentationDigits(nil, 12, '0', 7, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(queue) != 4 {
		t.Fatalf("queue len=%d, want 4", len(queue))
	}
	wantPos := []int{2, 7, 12, 17}
	wantDigit := []int{0, 0, '1', '2'}
	for i := range queue {
		if queue[i].PositionCode != wantPos[i] || queue[i].Target != 7 || queue[i].Digit != wantDigit[i] {
			t.Fatalf("entry %d=%+v, want pos=%d digit=%d", i, queue[i], wantPos[i], wantDigit[i])
		}
	}
}

func TestAppendNativePresentationDigitsBiasAndCameraGate(t *testing.T) {
	queue, err := AppendNativePresentationDigits(nil, 1234, 5, 2, true)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []int{6, 7, 8, 9} {
		if queue[i].Digit != want {
			t.Fatalf("digit %d=%d, want %d", i, queue[i].Digit, want)
		}
	}
	unchanged, err := AppendNativePresentationDigits(queue, 99, '0', 2, false)
	if err != nil || len(unchanged) != len(queue) {
		t.Fatalf("off-camera append changed queue: len=%d err=%v", len(unchanged), err)
	}
	if _, err := AppendNativePresentationDigits(nil, 1, 256, 0, true); err == nil {
		t.Fatal("invalid digit bias unexpectedly accepted")
	}
}

func TestAppendNativePresentationDigitsPreservesNativeLongValueRead(t *testing.T) {
	queue, err := AppendNativePresentationDigits(nil, 12345, '0', 1, true)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []int{'1', '2', '3', '4'} {
		if queue[i].Digit != want {
			t.Fatalf("long value digit %d=%d, want %d", i, queue[i].Digit, want)
		}
	}
}
