package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewAtkAnimRequiresNativeDelayPairing(t *testing.T) {
	t.Setenv("FD2_BATTLE_FPT", "2")
	g := &Game{
		figani: map[int][]*ebiten.Image{
			13: {ebiten.NewImage(1, 1), ebiten.NewImage(1, 1), ebiten.NewImage(1, 1)},
		},
		figaniDelays: map[int][]int{13: {1, 2, 1}},
	}
	a := g.newAtkAnim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 2, 0, 28, 8, 28, 0, true)
	if a == nil || a.figaniTimeline == nil {
		t.Fatal("paired FIGANI delay schedule did not create an attack presentation")
	}
	if a.bodyTicks != 8 || a.total != 16 || a.frameIndex != 0 {
		t.Fatalf("attack timeline body=%d total=%d frame=%d", a.bodyTicks, a.total, a.frameIndex)
	}
	if got, ok := a.figaniTimeline.FrameStart(2); !ok || got != 6 {
		t.Fatalf("frame 2 start=%d/%v, want 6/true", got, ok)
	}

	g.figaniDelays = nil
	if got := g.newAtkAnim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 2, 0, 28, 8, 28, 0, true); got != nil {
		t.Fatal("unpaired FIGANI PNGs received a guessed attack timeline")
	}
}

func TestNewAtkAnimLeavesNativeImpactDACUnwired(t *testing.T) {
	g := &Game{
		figani: map[int][]*ebiten.Image{
			13: {ebiten.NewImage(1, 1), ebiten.NewImage(1, 1), ebiten.NewImage(1, 1)},
		},
		figaniDelays: map[int][]int{13: {1, 2, 1}},
	}
	a := g.newAtkAnim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 2, 0, 28, 8, 28, 0, true)
	if a == nil {
		t.Fatal("paired FIGANI delay schedule did not create an attack presentation")
	}
	if a.nativeImpactRaw != nil || nativeImpactDACAllowed(a.nativeImpactRaw) {
		t.Fatal("attack animation inferred native DAC output without raw provenance")
	}
}

func TestNativeImpactDACRequiresRawFrameAndDamageProvenance(t *testing.T) {
	cases := []struct {
		name string
		raw  *nativeImpactDACInput
		want bool
	}{
		{name: "missing", raw: nil, want: false},
		{name: "frame flag missing", raw: &nativeImpactDACInput{damageStepComplete: true, rawOutput20: true}, want: false},
		{name: "damage step incomplete", raw: &nativeImpactDACInput{frameFlag: 1, rawOutput1C: true}, want: false},
		{name: "output absent", raw: &nativeImpactDACInput{frameFlag: 1, damageStepComplete: true}, want: false},
		{name: "first raw output", raw: &nativeImpactDACInput{frameFlag: 1, damageStepComplete: true, rawOutput20: true}, want: true},
		{name: "second raw output", raw: &nativeImpactDACInput{frameFlag: 1, damageStepComplete: true, rawOutput1C: true}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nativeImpactDACAllowed(tc.raw); got != tc.want {
				t.Fatalf("nativeImpactDACAllowed()=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestBattleImpactHPCommitsPostHitValueAtImpactBoundary(t *testing.T) {
	const impactStart = 33
	if got := battleImpactHP(impactStart-1, impactStart, 28, 8); got != 28 {
		t.Fatalf("pre-impact HP=%d, want 28", got)
	}
	if got := battleImpactHP(impactStart, impactStart, 28, 8); got != 8 {
		t.Fatalf("impact HP=%d, want committed post-hit value 8", got)
	}
	if got := battleImpactHP(impactStart+7, impactStart, 28, 8); got != 8 {
		t.Fatalf("post-impact HP=%d, want 8", got)
	}
}
