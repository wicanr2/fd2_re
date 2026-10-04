package battlepresent

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestNativePhysicalBodyHPAndZeroDelay(t *testing.T) {
	attack := &figani.Animation{Frames: []figani.Frame{
		{Delay: 0, RawByte4: 1, RawByte5: 2}, {Delay: 2}, {Delay: 1, RawByte4: 1, RawByte5: 3},
	}}
	idle := &figani.Animation{Frames: []figani.Frame{{Delay: 2}, {Delay: 3}}}
	strikes := []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 7}}, {Roll: battle.NativePhysicalRollResult{Damage: 20}}}
	plan, err := BuildNativePhysicalBodyPlan(attack, idle, 0, 12, strikes)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 6 || plan[0].HP != 9 || len(plan[0].Presents) != 0 || plan[2].HP != 5 || plan[3].HP != 0 || plan[5].HP != 0 {
		t.Fatalf("HP split / zero delay / overkill: %+v", plan)
	}
	// 零 delay 的命中仍設定phase與fill，直到下個實際present才消費。
	first := plan[1].Presents[0]
	if first.DX != 14 || first.DY != 10 || first.OpaqueFill != 33 || plan[1].Presents[1].OpaqueFill != -1 {
		t.Fatalf("zero-delay hit lost first actual impact: %+v", first)
	}
	// 第一揮三次idle呈現後是frame1/counter1，第二揮不重置。
	if plan[4].Presents[0].IdleFrame != 1 || plan[4].Presents[1].IdleFrame != 1 || plan[5].Presents[0].IdleFrame != 0 {
		t.Fatal("multi-strike idle counter was reset")
	}
}

func TestNativePhysicalBodyMissAndPulseGates(t *testing.T) {
	attack := &figani.Animation{HeaderByte1: 1, HeaderByte2: 1, Frames: []figani.Frame{
		{Delay: 1}, {Delay: 2, RawByte4: 1, RawByte5: 4},
	}}
	idle := &figani.Animation{Frames: []figani.Frame{{Delay: 1}}}
	for _, missed := range []bool{true, false} {
		roll := battle.NativePhysicalRollResult{Missed: missed, Crit: true, Status: true}
		p, err := BuildNativePhysicalBodyPlan(attack, idle, 2, 80, []battle.NativePhysicalStrike{{Roll: roll}})
		if err != nil {
			t.Fatal(err)
		}
		if len(p) != 1 || p[0].Frame != 1 || p[0].HP != 80 || p[0].Presents[0].DY != 0 || !p[0].Presents[0].StatusPulse || !p[0].Presents[1].CritPulse {
			t.Fatalf("header / pulse gate: %+v", p)
		}
		if missed && (p[0].Cue != 0 || p[0].Presents[0].DX != 0 || p[0].Presents[0].OpaqueFill != -1) {
			t.Fatal("MISS received displacement or lost cue0")
		}
		if !missed && (p[0].Cue != 4 || p[0].Presents[0].DX != -14 || p[0].Presents[0].OpaqueFill != 33) {
			t.Fatal("nonzero raw side lost signed impact")
		}
	}
	// 分母統計全部descriptors，包含header2之前；raw4=2不走pulse。
	attack.Frames[0].RawByte4 = 1
	attack.Frames[1].RawByte4 = 2
	p, err := BuildNativePhysicalBodyPlan(attack, idle, 0, 80, []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 9, Crit: true, Status: true}}})
	if err != nil || p[0].HP != 76 || p[0].Presents[0].CritPulse || p[0].Presents[0].StatusPulse {
		t.Fatal("raw4 / complete denominator gate changed")
	}
}

func TestNativePhysicalBodyIdleZeroWrap(t *testing.T) {
	attack := &figani.Animation{Frames: []figani.Frame{{Delay: 255}, {Delay: 2}}}
	idle := &figani.Animation{Frames: []figani.Frame{{Delay: 0}, {Delay: 1}}}
	p, err := BuildNativePhysicalBodyPlan(attack, idle, 0, 30, []battle.NativePhysicalStrike{{}})
	if err != nil {
		t.Fatal(err)
	}
	if p[0].Presents[254].IdleFrame != 0 || p[1].Presents[0].IdleFrame != 0 || p[1].Presents[1].IdleFrame != 1 {
		t.Fatal("idle delay0 must advance after 256 presents")
	}
}

func TestNativePhysicalBodyRawLayerOrderAndClipping(t *testing.T) {
	frame := func(value byte) figani.Frame {
		return figani.Frame{X: 319, Y: 199, Width: 2, Height: 1, Pixels: []byte{value, value}, Mask: []byte{255, 255}}
	}
	for _, side := range []byte{0, 1, 2} {
		for _, bit := range []byte{0, 1} {
			attack, idle := frame(10), frame(20)
			attack.RawByte7 = bit
			out, err := ComposeNativePhysicalBodyPresent(make([]byte, 64000), attack, idle, side, NativePhysicalBodyPresent{OpaqueFill: -1})
			if err != nil {
				t.Fatal(err)
			}
			want := byte(10)
			if (side == 0 && bit == 0) || (side != 0 && bit != 0) {
				want = 20
			}
			if out[63999] != want || out[63680] != 0 {
				t.Fatalf("raw side=%d bit=%d: edge=%d", side, bit, out[63999])
			}
		}
	}
}
