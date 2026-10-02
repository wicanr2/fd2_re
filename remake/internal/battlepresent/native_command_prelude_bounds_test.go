package battlepresent

import (
	"bytes"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestCommandPreludeOriginalBottomRowBothSides(t *testing.T) {
	if _, err := os.Stat("../../../org_game/炎龍騎士團/FLAME2/FIGANI.DAT"); os.IsNotExist(err) {
		t.Skip("player-provided FIGANI.DAT is absent")
	}
	root := "../../generated-assets/fd2-original-b97caf22/animations"
	tall, err := figani.LoadSeparatedResource(root, 51)
	if err != nil {
		t.Fatal(err)
	}
	short, err := figani.LoadSeparatedResource(root, 309)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name          string
		side          byte
		actor, target figani.Frame
	}{
		{"enemy-target", 0, short.Frames[0], tall.Frames[0]},
		{"enemy-actor", 0, tall.Frames[0], short.Frames[0]},
		{"own-target", 2, short.Frames[0], tall.Frames[0]},
		{"own-actor", 2, tall.Frames[0], short.Frames[0]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := bytes.Repeat([]byte{9}, 320*200)
			before := append([]byte(nil), base...)
			platform := fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 7}}
			frames, err := BuildNativeCommandPreludeFrames(NativeCommandPreludeInput{
				Base: base, ActorIdle: tc.actor, FirstTargetIdle: tc.target, Platform: &platform,
				RawSide: tc.side, Mode: 0, BaselineDAC: command24PreludeDAC(60),
			})
			if err != nil || len(frames) != 9 || frames[8].Stage != 0 {
				t.Fatalf("完整九段未建立：frames=%d err=%v", len(frames), err)
			}
			want := append([]byte(nil), base...)
			platform.X, platform.Y = 164, 157
			if tc.side != 0 {
				if err := tc.target.BlitAt(want, 320); err != nil {
					t.Fatal(err)
				}
			}
			if err := platform.Blit(want, 320, -1); err != nil {
				t.Fatal(err)
			}
			if err := tc.actor.BlitAt(want, 320); err != nil {
				t.Fatal(err)
			}
			if tc.side == 0 {
				if err := tc.target.BlitAt(want, 320); err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(frames[8].Pixels, want) || !bytes.Equal(base, before) {
				t.Fatal("末段640-stride viewport與320-wide完整原生角色畫面不相同，或輸入被改寫")
			}
		})
	}
}
