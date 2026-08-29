package fdother

import (
	"bytes"
	"os"
	"testing"
)

func TestSeparatedEvent61FramesMatchOriginal(t *testing.T) {
	const root = "../../generated-assets/fd2-original-b97caf22/animations/fdother_045_event61"
	const archive = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	want, err := DecodeResource(archive, 45)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedEvent61Frames(root)
	if err != nil {
		t.Skipf("separated event61 pack is absent: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("frame count=%d want=%d", len(got), len(want))
	}
	for index := range want {
		wantIndexed, wantMask, err := want[index].IndexedLayers()
		if err != nil {
			t.Fatalf("original frame %d: %v", index, err)
		}
		if got[index].X != want[index].X || got[index].Y != want[index].Y ||
			got[index].Width != want[index].Width || got[index].Height != want[index].Height ||
			!bytes.Equal(got[index].Indexed, wantIndexed) ||
			!bytes.Equal(got[index].Mask, wantMask) {
			t.Fatalf("event61 frame %d differs", index)
		}
	}
}

func TestSeparatedEvent61FramesFailClosedWithoutPack(t *testing.T) {
	if _, err := LoadSeparatedEvent61Frames(t.TempDir()); err == nil {
		t.Fatal("incomplete event61 pack was accepted")
	}
}

func TestSeparatedEndingPrefixFramesMatchOriginal(t *testing.T) {
	const root = "../../generated-assets/fd2-original-b97caf22/animations/fdother_054_ending_prefix"
	const archive = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	want, err := DecodeResource(archive, 54)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedEndingPrefixFrames(root)
	if err != nil {
		t.Skipf("separated ending prefix pack is absent: %v", err)
	}
	if len(got) != len(want) || len(got) != nativeEndingPrefixFrameCount {
		t.Fatalf("frame count=%d want=%d", len(got), len(want))
	}
	for index := range want {
		wantIndexed, wantMask, err := want[index].IndexedLayers()
		if err != nil {
			t.Fatalf("original frame %d: %v", index, err)
		}
		if got[index].X != want[index].X || got[index].Y != want[index].Y ||
			got[index].Width != want[index].Width || got[index].Height != want[index].Height ||
			!bytes.Equal(got[index].Indexed, wantIndexed) ||
			!bytes.Equal(got[index].Mask, wantMask) {
			t.Fatalf("ending prefix frame %d differs", index)
		}
	}
}

func TestSeparatedEndingPrefixFramesFailClosedWithoutPack(t *testing.T) {
	if _, err := LoadSeparatedEndingPrefixFrames(t.TempDir()); err == nil {
		t.Fatal("incomplete ending prefix pack was accepted")
	}
}

func TestSeparatedCh20SkyKeyFramesMatchOriginal(t *testing.T) {
	const root = "../../generated-assets/fd2-original-b97caf22/animations/fdother_034_ch20_sky_key"
	const archive = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	want, err := DecodeResource(archive, 34)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedCh20SkyKeyFrames(root)
	if err != nil {
		t.Skipf("separated ch20 sky-key pack is absent: %v", err)
	}
	if len(got) != len(want) || len(got) != nativeCh20SkyKeyFrameCount {
		t.Fatalf("frame count=%d want=%d", len(got), len(want))
	}
	for index := range want {
		wantIndexed, wantMask, err := want[index].IndexedLayers()
		if err != nil {
			t.Fatalf("original frame %d: %v", index, err)
		}
		if got[index].X != want[index].X || got[index].Y != want[index].Y ||
			got[index].Width != want[index].Width || got[index].Height != want[index].Height ||
			!bytes.Equal(got[index].Indexed, wantIndexed) ||
			!bytes.Equal(got[index].Mask, wantMask) {
			t.Fatalf("ch20 sky-key frame %d differs", index)
		}
	}
}

func TestSeparatedCh20SkyKeyFramesFailClosedWithoutPack(t *testing.T) {
	if _, err := LoadSeparatedCh20SkyKeyFrames(t.TempDir()); err == nil {
		t.Fatal("incomplete ch20 sky-key pack was accepted")
	}
}

func TestSeparatedPendingCode1FramesMatchOriginal(t *testing.T) {
	const root = "../../generated-assets/fd2-original-b97caf22/animations/fdother_079_pending_code1"
	const archive = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(archive); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	want, err := DecodeResource(archive, 79)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedPendingCode1Frames(root)
	if err != nil {
		t.Skipf("separated pending-code-1 pack is absent: %v", err)
	}
	if len(got) != len(want) || len(got) != nativePendingCode1FrameCount {
		t.Fatalf("frame count=%d want=%d", len(got), len(want))
	}
	for index := range want {
		wantIndexed, wantMask, err := want[index].IndexedLayers()
		if err != nil {
			t.Fatalf("original frame %d: %v", index, err)
		}
		if got[index].X != want[index].X || got[index].Y != want[index].Y ||
			got[index].Width != want[index].Width || got[index].Height != want[index].Height ||
			!bytes.Equal(got[index].Indexed, wantIndexed) ||
			!bytes.Equal(got[index].Mask, wantMask) {
			t.Fatalf("pending-code-1 frame %d differs", index)
		}
	}
}

func TestSeparatedPendingCode1FramesFailClosedWithoutPack(t *testing.T) {
	if _, err := LoadSeparatedPendingCode1Frames(t.TempDir()); err == nil {
		t.Fatal("incomplete pending-code-1 pack was accepted")
	}
}
