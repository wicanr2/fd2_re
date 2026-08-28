package ending

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func loadOriginalMontageTailPlayer(t *testing.T) (*MontageTailPlayer, MontageTailAssets) {
	t.Helper()
	const gameRoot = "../../../org_game/炎龍騎士團/FLAME2"
	for _, name := range []string{"FDOTHER.DAT", "TAI.DAT", "BG.DAT"} {
		if _, err := os.Stat(filepath.Join(gameRoot, name)); os.IsNotExist(err) {
			t.Skip("player-provided ending resources are unavailable")
		} else if err != nil {
			t.Fatal(err)
		}
	}
	tail, err := LoadMontageTail(filepath.Join("..", "..", "assets", "endings", "native_2c194_tail.json"))
	if err != nil {
		t.Fatal(err)
	}
	assets, err := LoadMontageTailAssetsArchive(*tail, filepath.Join(gameRoot, "FDOTHER.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	sets, err := LoadMontageTailVisualSets(*tail, MontageTailVisualPaths{
		SurfaceRoot:   "../../generated-assets/fd2-original-b97caf22/surfaces",
		AnimationRoot: "../../generated-assets/fd2-original-b97caf22/animations",
	})
	if err != nil {
		t.Fatal(err)
	}
	player, err := NewMontageTailPlayer(*tail, assets, sets, NewIndexedCompositor())
	if err != nil {
		t.Fatal(err)
	}
	return player, assets
}

func TestMontageTailPlayerUsesAllTwentySourceTransactionsThenHoldsFinal(t *testing.T) {
	player, assets := loadOriginalMontageTailPlayer(t)
	for index, set := range player.VisualSets {
		for name, animation := range map[string]*figani.Animation{
			"record0 base": set.Record0FIGANIBase, "record0 aux": set.Record0FIGANIAux,
			"record1 base": set.Record1FIGANIBase, "record1 aux": set.Record1FIGANIAux,
		} {
			if animation.HeaderByte1 != 0 || int(animation.HeaderByte2) != len(animation.Frames) {
				t.Fatalf("tail entry %d %s header=%d/%d frames=%d", index, name, animation.HeaderByte1, animation.HeaderByte2, len(animation.Frames))
			}
		}
	}
	seenSegments := make(map[int]bool)
	var hashes [][32]byte
	positiveWaits := 0
	for steps := 0; steps < 4096 && !player.Ready(); steps++ {
		beforeSegment := player.Segment
		beforePhase := player.Phase
		if err := player.Step(); err != nil {
			t.Fatalf("tail step %d segment=%d phase=%s: %v", steps, beforeSegment, beforePhase, err)
		}
		if beforePhase != MontageTailPhaseIntro && beforePhase != MontageTailPhaseFinal {
			seenSegments[beforeSegment] = true
		}
		if player.DelayTicks > 0 {
			positiveWaits++
		}
		hashes = append(hashes, sha256.Sum256(player.Compositor.VGA))
	}
	if len(hashes) == 0 || positiveWaits == 0 {
		t.Fatal("tail intro did not expose its source-proven wait")
	}
	if !player.Ready() || player.Segment != 20 || len(seenSegments) != 20 || positiveWaits < 60 {
		t.Fatalf("tail completion ready=%v segment=%d seen=%d waits=%d", player.Ready(), player.Segment, len(seenSegments), positiveWaits)
	}
	if len(hashes) < 100 {
		t.Fatalf("tail produced only %d indexed presentations", len(hashes))
	}
	final := NewIndexedCompositor()
	copy(final.Palette[:], assets.LoopPalette)
	copy(final.Baseline[:], assets.LoopPalette)
	final.baselineKnown = true
	if err := assets.PresentFinal(final); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(player.Compositor.VGA, final.VGA) || !bytes.Equal(player.Compositor.Palette[:], final.Palette[:]) {
		t.Fatal("tail player did not finish on the source-derived FDOTHER#59 image")
	}
	before := append([]byte(nil), player.Compositor.VGA...)
	if err := player.Step(); err == nil || !bytes.Equal(before, player.Compositor.VGA) {
		t.Fatalf("completed tail accepted another step or mutated terminal frame: err=%v", err)
	}
}

func TestMontageTailPlayerIsDeterministicForOriginalAssets(t *testing.T) {
	first, _ := loadOriginalMontageTailPlayer(t)
	second, _ := loadOriginalMontageTailPlayer(t)
	for steps := 0; steps < 4096 && (!first.Ready() || !second.Ready()); steps++ {
		if first.Ready() != second.Ready() || first.Phase != second.Phase || first.Segment != second.Segment || first.Frame != second.Frame {
			t.Fatalf("players diverged before step %d", steps)
		}
		if err := first.Step(); err != nil {
			t.Fatal(err)
		}
		if err := second.Step(); err != nil {
			t.Fatal(err)
		}
		if first.DelayTicks != second.DelayTicks || !bytes.Equal(first.Compositor.VGA, second.Compositor.VGA) ||
			!bytes.Equal(first.Compositor.Palette[:], second.Compositor.Palette[:]) {
			t.Fatalf("players diverged after step %d", steps)
		}
	}
	if !first.Ready() || !second.Ready() {
		t.Fatal("deterministic players exceeded bounded completion budget")
	}
}

func TestMontageTailPlayerFailsClosedBeforePartialPlayback(t *testing.T) {
	tail := MontageTail{Loop: MontageTailLoop{Count: 1}, RawTables: MontageTailRawTable{
		Record0Byte7: []int{1}, Record1Byte7: []int{2}, Global540FF: []int{3},
	}}
	if player, err := NewMontageTailPlayer(tail, MontageTailAssets{}, nil, NewIndexedCompositor()); err == nil || player != nil {
		t.Fatalf("incomplete tail produced player=%#v err=%v", player, err)
	}
}

func solidTailFDOTHERFrame(colour byte) fdother.Frame {
	return fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, colour}}
}

func solidTailFIGANIFrame(colour, raw4, raw5, delay, raw7 byte) figani.Frame {
	return figani.Frame{
		X: 10, Y: 10, Width: 1, Height: 1,
		Pixels: []byte{colour}, Mask: []byte{1}, Delay: int(delay),
		RawByte4: raw4, RawByte5: raw5, RawByte7: raw7,
	}
}

func TestMontageTailNativePairUsesRawInnerCountZOrderAndBaseScheduler(t *testing.T) {
	auxiliary := &figani.Animation{HeaderByte2: 1, Frames: []figani.Frame{
		solidTailFIGANIFrame(9, 0, 42, 2, 1),
	}}
	base := &figani.Animation{HeaderByte2: 2, Frames: []figani.Frame{
		solidTailFIGANIFrame(7, 0, 0, 1, 0),
		solidTailFIGANIFrame(8, 0, 0, 1, 0),
	}}
	set := MontageTailVisualSet{
		TAI: solidTailFDOTHERFrame(2), BG: solidTailFDOTHERFrame(3),
		Record0FIGANIAux: auxiliary, Record1FIGANIBase: base,
	}
	p := &MontageTailPlayer{
		Compositor: NewIndexedCompositor(), Segment: 0,
		Entries: []MontageTailEntry{{Record0Byte6: 2}},
	}
	for wantBase := byte(7); wantBase <= 8; wantBase++ {
		presented, done, err := p.stepNativePair(set, auxiliary, base, true, false)
		if err != nil || !presented || done || p.DelayTicks != 1 || p.SoundMarker != 42 {
			t.Fatalf("pair step base=%d presented=%v done=%v delay=%d marker=%d err=%v", wantBase, presented, done, p.DelayTicks, p.SoundMarker, err)
		}
		// side!=0 and raw +7 bit0=1 draws auxiliary first, then paired base.
		if got := p.Compositor.VGA[10*Width+10]; got != wantBase {
			t.Fatalf("paired base scheduler pixel=%d want=%d", got, wantBase)
		}
	}
	if p.Frame != 1 || p.Inner != 0 || p.BaseFrame != 0 {
		t.Fatalf("pair state frame=%d inner=%d base=%d", p.Frame, p.Inner, p.BaseFrame)
	}
}

func TestMontageTailSecondPairStopsAfterLastRawByte4Frame(t *testing.T) {
	auxiliary := &figani.Animation{HeaderByte2: 3, Frames: []figani.Frame{
		solidTailFIGANIFrame(9, 0, 0, 1, 0),
		solidTailFIGANIFrame(10, 1, 0, 1, 0),
		solidTailFIGANIFrame(11, 0, 0, 1, 0),
	}}
	base := &figani.Animation{HeaderByte2: 1, Frames: []figani.Frame{
		solidTailFIGANIFrame(7, 0, 0, 1, 0),
	}}
	set := MontageTailVisualSet{TAI: solidTailFDOTHERFrame(2), BG: solidTailFDOTHERFrame(3)}
	p := &MontageTailPlayer{
		Compositor: NewIndexedCompositor(), Segment: 0,
		Entries: []MontageTailEntry{{Record1Byte6: 0}},
	}
	if presented, done, err := p.stepNativePair(set, auxiliary, base, false, true); err != nil || !presented || done {
		t.Fatalf("first terminal pair step presented=%v done=%v err=%v", presented, done, err)
	}
	if presented, done, err := p.stepNativePair(set, auxiliary, base, false, true); err != nil || !presented || !done {
		t.Fatalf("last +4 terminal pair step presented=%v done=%v err=%v", presented, done, err)
	}
	if got := p.Compositor.VGA[20*Width+24]; got != 33 || p.EffectStep != 4 {
		t.Fatalf("raw +4 displacement/opaque override pixel=%d effectStep=%d, want 33/4", got, p.EffectStep)
	}
	if p.Frame != 2 {
		t.Fatalf("terminal pair consumed frame index=%d, want stop before frame2", p.Frame)
	}
}

func TestMontageTailRejectsNonzeroPreludeBeforePlayback(t *testing.T) {
	tail := MontageTail{
		Loop:      MontageTailLoop{Count: 1},
		RawTables: MontageTailRawTable{Record0Byte7: []int{1}, Record1Byte7: []int{2}, Global540FF: []int{3}},
	}
	plans, err := tail.PlanVisualResources()
	if err != nil {
		t.Fatal(err)
	}
	frame := solidTailFIGANIFrame(7, 0, 0, 1, 0)
	valid := &figani.Animation{HeaderByte2: 1, Frames: []figani.Frame{frame}}
	prelude := &figani.Animation{HeaderByte1: 1, HeaderByte2: 1, Frames: []figani.Frame{frame}}
	set := MontageTailVisualSet{
		Plan: plans[0], TAI: solidTailFDOTHERFrame(2), BG: solidTailFDOTHERFrame(3),
		Record0FIGANIBase: valid, Record0FIGANIAux: prelude,
		Record1FIGANIBase: valid, Record1FIGANIAux: valid,
	}
	assets := MontageTailAssets{
		LoopPalette: make([]byte, 768), Intro: solidTailFDOTHERFrame(1),
		LoopFrames: []fdother.Frame{solidTailFDOTHERFrame(4)}, Final: solidTailFDOTHERFrame(5),
	}
	if player, err := NewMontageTailPlayer(tail, assets, []MontageTailVisualSet{set}, NewIndexedCompositor()); err == nil || player != nil {
		t.Fatalf("nonzero prelude entered tail player=%#v err=%v", player, err)
	}
}
