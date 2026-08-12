package ending

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func loadOriginalMontageTailPlayer(t *testing.T) (*MontageTailPlayer, MontageTailAssets) {
	t.Helper()
	const gameRoot = "../../../org_game/炎龍騎士團/FLAME2"
	for _, name := range []string{"FDOTHER.DAT", "TAI.DAT", "BG.DAT", "FIGANI.DAT"} {
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
	assets, err := LoadMontageTailAssets(*tail, filepath.Join(gameRoot, "FDOTHER.DAT"))
	if err != nil {
		t.Fatal(err)
	}
	sets, err := LoadMontageTailVisualSets(*tail, MontageTailVisualPaths{
		TAI: filepath.Join(gameRoot, "TAI.DAT"), BG: filepath.Join(gameRoot, "BG.DAT"),
		FIGANI: filepath.Join(gameRoot, "FIGANI.DAT"),
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
