package fdother

import "testing"

func TestNativeCh28PostOverlayFramesPreserveTwoSixFrameLoops(t *testing.T) {
	frames := NativeCh28PostOverlayFrames()
	if len(frames) != 12 {
		t.Fatalf("frames=%d want 12", len(frames))
	}
	for index, frame := range frames {
		if frame.Entry != NativeCh28PostLMIFirst+index {
			t.Fatalf("frame%d entry=%#x", index, frame.Entry)
		}
		wantPhase := "overlay_a"
		if index >= 6 {
			wantPhase = "overlay_b"
		}
		if frame.Phase != wantPhase {
			t.Fatalf("frame%d phase=%q want %q", index, frame.Phase, wantPhase)
		}
	}
}

func TestNativeCh28PostOverlayOriginMatchesDirectAddressExpression(t *testing.T) {
	got, err := NativeCh28PostOverlayOrigin(10, 8, 9, 7)
	if err != nil {
		t.Fatal(err)
	}
	want := NativeCh28PostViewBase - 6*NativeCh28PostStride
	if got != want {
		t.Fatalf("origin=%#x want %#x", got, want)
	}
	if _, err := NativeCh28PostOverlayOrigin(-100, -100, 0, 0); err == nil {
		t.Fatal("off-surface origin was accepted")
	}
}

func TestBlitNativeCh28PostOverlayUsesTransparent4E85BRule(t *testing.T) {
	work := make([]byte, NativeCh28PostWorkSize)
	origin, err := NativeCh28PostOverlayOrigin(10, 8, 9, 7)
	if err != nil {
		t.Fatal(err)
	}
	work[origin], work[origin+1] = 7, 8
	entry := LMI1Entry{Width: 2, Height: 1, Pixels: []byte{0, 9}}
	if err := BlitNativeCh28PostOverlay(entry, work, origin); err != nil {
		t.Fatal(err)
	}
	if work[origin] != 7 || work[origin+1] != 9 {
		t.Fatalf("pixels=%v want [7 9]", work[origin:origin+2])
	}
}

func TestValidateNativeCh28PostLMIRequiresExactRawEntryRange(t *testing.T) {
	entries := make([]LMI1Entry, NativeCh28PostLMILast+1)
	for index := NativeCh28PostLMIFirst; index <= NativeCh28PostLMILast; index++ {
		entries[index] = LMI1Entry{Width: 1, Height: 1, Pixels: []byte{byte(index)}}
	}
	if err := ValidateNativeCh28PostLMI(entries); err != nil {
		t.Fatal(err)
	}
	entries[NativeCh28PostLMIFirst+2] = LMI1Entry{}
	if err := ValidateNativeCh28PostLMI(entries); err == nil {
		t.Fatal("malformed required entry was accepted")
	}
}
