package battlepresent

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestComposeNativeCommand6TargetFramePreservesModeOrder(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 10), HeaderByte2: 10}
	for index := range effect.Frames {
		effect.Frames[index] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}}
	}
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}}
	frame := figani.NativeCommand6TargetFrame{
		Mode4: []figani.NativeCommand6Layer{{Mode: 4, Channel: 0, Frame: 4}},
		Mode5: []figani.NativeCommand6Layer{{Mode: 5, Channel: 0, Frame: 5}},
	}
	got, err := ComposeNativeCommand6TargetFrame(make([]byte, 320*200), target, target, effect, frame, figani.NativeCommand6TargetDisplayFrame{Shader: -1})
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 6 {
		t.Fatalf("composed pixel=%d want final mode5 frame5", got[0])
	}
}

func TestComposeNativeCommand6TargetFrameRejectsInvalidLayer(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, 10), HeaderByte2: 10}
	frame := figani.NativeCommand6TargetFrame{Mode4: []figani.NativeCommand6Layer{{Frame: 10}}}
	if _, err := ComposeNativeCommand6TargetFrame(make([]byte, 320*200), figani.Frame{}, figani.Frame{}, effect, frame, figani.NativeCommand6TargetDisplayFrame{Shader: -1}); err == nil {
		t.Fatal("out-of-range command6 layer accepted")
	}
}

func TestComposeNativeCommand6OrbitFramePreservesLayerOrder(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, figani.NativeCommand6EffectFrameCount), HeaderByte2: figani.NativeCommand6EffectFrameCount}
	effect.Frames[4] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}}}
	frame := figani.NativeCommand6OrbitFrame{
		First:  []figani.NativeCommand6Layer{{Mode: 1, Frame: 4, X: 1, Y: 1}},
		Second: []figani.NativeCommand6Layer{{Mode: 2, Frame: 4, X: 2, Y: 1}},
	}
	got, err := ComposeNativeCommand6OrbitFrame(make([]byte, 320*200), actor, nil, effect, frame)
	if err != nil {
		t.Fatal(err)
	}
	if got[1*320+1] != 7 || got[1*320+2] != 7 {
		t.Fatalf("orbit pixels=%d,%d", got[1*320+1], got[1*320+2])
	}
	if got[0] != 9 {
		t.Fatalf("orbit actor pixel=%d", got[0])
	}
}

func TestComposeNativeCommand6TailOrbitRequiresAndDrawsTarget(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, figani.NativeCommand6EffectFrameCount)}
	actor := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}}}
	target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}, X: 2}
	frame := figani.NativeCommand6OrbitFrame{DrawTarget: true}
	if _, err := ComposeNativeCommand6OrbitFrame(make([]byte, 320*200), actor, nil, effect, frame); err == nil {
		t.Fatal("tail without target accepted")
	}
	got, err := ComposeNativeCommand6OrbitFrame(make([]byte, 320*200), actor, &target, effect, frame)
	if err != nil || got[2] != 9 {
		t.Fatalf("tail target pixel=%d err=%v", got[2], err)
	}
}

func TestComposeNativeCommand6TransitionFrameUsesLastActorAndSelectedTarget(t *testing.T) {
	effect := &figani.Animation{Frames: make([]figani.Frame, figani.NativeCommand6EffectFrameCount)}
	actor := &figani.Animation{Frames: []figani.Frame{
		{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}},
		{Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}},
	}}
	target := &figani.Animation{Frames: []figani.Frame{{Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}}}
	got, err := ComposeNativeCommand6TransitionFrame(make([]byte, 320*200), actor, target, effect, figani.NativeCommand6TransitionFrame{TargetOffsetX: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 7 || got[2] != 9 {
		t.Fatalf("transition actor/target pixels=%d,%d", got[0], got[2])
	}
}

// #151：直接解碼固定原始FDOTHER，對照原版0x2A300-byte work配置。
// 這是資產／compositor回歸，不當作原版一般玩家逐幀收據。
func TestComposeNativeCommand6RealResourcesMatchNativeWorkAllocation(t *testing.T) {
	path := "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("原版資產未提供")
	}
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce" {
		t.Fatalf("原版FDOTHER雜湊不符：%s", got)
	}
	for _, side := range []byte{0, 1} {
		resource := 32
		if side == 0 {
			resource = 33
		}
		effect, err := figani.DecodeResource(path, resource)
		if err != nil {
			t.Fatal(err)
		}
		schedule, err := figani.BuildNativeCommand6PresentationSchedule(side, effect)
		if err != nil {
			t.Fatal(err)
		}
		points := figani.NativeCommand6TargetCoordinates(figani.NativeCommand6Coordinates(36, schedule.BaseByte), 42)
		sequence, err := figani.BuildNativeCommand6TargetSequence(schedule, points, side)
		if err != nil {
			t.Fatal(err)
		}
		target := figani.Frame{Width: 1, Height: 1, Pixels: []byte{99}, Mask: []byte{1}}
		actor := figani.Frame{X: 300, Width: 1, Height: 1, Pixels: []byte{42}, Mask: []byte{1}}
		for step, planned := range sequence {
			// 獨立參照：用既有FIGANI嚴格blit寫原版大小，依原始viewport擷取。
			work := make([]byte, 0x2A300)
			var referenceErr error
			blit := func(frame figani.Frame, x, y int) {
				frame.X += 160 + x
				frame.Y += 30 + y
				if err := frame.BlitAt(work, 640); err != nil {
					referenceErr = err
				}
			}
			for _, layer := range planned.Mode4 {
				blit(effect.Frames[layer.Frame], layer.X, layer.Y)
			}
			blit(actor, 0, 0)
			blit(target, 0, 0)
			for _, layer := range planned.Mode5 {
				blit(effect.Frames[layer.Frame], layer.X, layer.Y)
			}
			expected := make([]byte, 320*200)
			for y := 0; y < 200; y++ {
				copy(expected[y*320:(y+1)*320], work[0x4BA0+y*640:0x4BA0+y*640+320])
			}
			got, err := ComposeNativeCommand6TargetFrame(make([]byte, 320*200), actor, target, effect, planned, figani.NativeCommand6TargetDisplayFrame{Shader: -1})
			if referenceErr != nil {
				// #154：負列位置仍未知，保留零partial-output拒收，不稱已修正。
				if err == nil || got != nil {
					t.Fatalf("原版resource%d step%d未知邊界被發布", resource, step)
				}
				continue
			}
			if err != nil {
				t.Fatalf("原版resource%d side%d target step%d：%v", resource, side, step, err)
			}
			for i, value := range expected {
				if got[i] != value {
					t.Fatalf("resource%d step%d viewport pixel%d：%d != %d", resource, step, i, got[i], value)
				}
			}
		}
		// 完整兩目標 owner：#33 必須可預建，#32 的 #154 負列仍零發布。
		base := make([]byte, 320*200)
		stages := make([][]byte, figani.NativeCommand6DamageStages+1)
		for i := range stages {
			stages[i] = base
		}
		target.Delay = 1
		idle := &figani.Animation{Frames: []figani.Frame{target}}
		all, err := BuildNativeCommand6EffectSequence(NativeCommand6EffectInput{
			FrontBase: base, TailBase: base, TargetBases: [][][]byte{stages, stages}, TransitionBases: [][]byte{base},
			ActorEffect: idle, TargetIdle: []*figani.Animation{idle, idle}, Effect: effect, Schedule: schedule, RawSide: side, TargetHits: []bool{false, false}, TargetNumericRNG: []uint16{1, 1},
		})
		if side == 0 {
			if err != nil || len(all.Targets) != 2 || len(all.Targets[1].Frames) != 12 || len(all.Transitions) != 1 {
				t.Fatalf("原始#33完整兩目標預建失敗：%v", err)
			}
		} else if err == nil || len(all.Front) != 0 || len(all.Targets) != 0 || len(all.Transitions) != 0 || len(all.Tail) != 0 {
			t.Fatalf("原始#32負列發布部分sequence：%+v err=%v", all, err)
		}
		for _, pixel := range base {
			if pixel != 0 {
				t.Fatal("原始資產預建改寫caller base")
			}
		}
	}
}

func TestNativeCommandWorkBoundsMatchAllocation(t *testing.T) {
	work := make([]byte, 0x2A300)
	last := figani.Frame{X: 479, Y: 239, Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}
	if err := blitCommand0WorkFrame(work, last, 0, 0); err != nil {
		t.Fatal(err)
	}
	if work[len(work)-1] != 7 {
		t.Fatal("原版work最後一byte未寫入")
	}
	for _, frame := range []figani.Frame{
		{X: 479, Y: 240, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}},
		{X: 480, Y: 239, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}},
		{X: 479, Y: 239, Width: 1, Height: 1, Pixels: []byte{9}},
	} {
		if err := blitCommand0WorkFrame(work, frame, 0, 0); err == nil {
			t.Fatal("真正越界或malformed影格未拒收")
		}
		if work[len(work)-1] != 7 {
			t.Fatal("拒收影格發布部分像素")
		}
	}
}
