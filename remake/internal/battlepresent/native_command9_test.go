package battlepresent

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// 用獨立320×200遮罩覆寫器重組來源，避免以同一work-buffer helper自證。
func TestNativeCommand9SourcesRecompose119Frames(t *testing.T) {
	actor, target, effect := command9PresentAnimation(1, 1), command9PresentAnimation(3, 2), command9PresentAnimation(31, 0)
	for _, animation := range []*figani.Animation{actor, target, effect} {
		for i := range animation.Frames {
			f := &animation.Frames[i]
			f.X, f.Y, f.Width, f.Height = 110, 55, 4, 2
			f.Pixels, f.Mask = bytes.Repeat([]byte{byte(i + 40)}, 8), []byte{1, 0, 1, 1, 1, 1, 0, 1}
		}
	}
	actor.Frames[0].Pixels = bytes.Repeat([]byte{200}, 8)
	for i := range target.Frames {
		target.Frames[i].Pixels = bytes.Repeat([]byte{byte(i + 100)}, 8)
	}
	effect.Frames[30].Mask = make([]byte, 8)
	bases := make([][]byte, 21)
	for i := range bases {
		bases[i] = bytes.Repeat([]byte{byte(i + 1)}, 64000)
	}
	schedule, err := figani.BuildNativeCommand9AISchedule(0, effect)
	if err != nil {
		t.Fatal(err)
	}
	seq, err := BuildNativeCommand9AISequence(NativeCommand9AIInput{BeforeBase: bases[0], AfterBase: bases[20], TargetBases: bases,
		Actor: actor, Target: target, Effect: effect, Schedule: schedule})
	if err != nil {
		t.Fatal(err)
	}
	count, stages, sub1, sub2, blank := 0, 0, 0, 0, 0
	for _, group := range [][]NativeCommand9RenderedFrame{seq.SlideIn, seq.Front, seq.Target, seq.Tail, seq.SlideOut} {
		for _, frame := range group {
			s := frame.Sources
			out := append([]byte(nil), s.Base...)
			blit := func(f figani.Frame, dx int) {
				for y := 0; y < f.Height; y++ {
					for x := 0; x < f.Width; x++ {
						at, px, py := y*f.Width+x, f.X+x+dx, f.Y+y
						if f.Mask[at] != 0 && px >= 0 && px < 320 && py >= 0 && py < 200 {
							out[py*320+px] = f.Pixels[at]
						}
					}
				}
			}
			for _, e := range s.Before {
				blit(effect.Frames[e.Frame], e.X)
			}
			if s.ActorFrame >= 0 {
				blit(actor.Frames[s.ActorFrame], s.ActorOffsetX)
			}
			blit(target.Frames[s.TargetFrame], s.TargetOffsetX)
			for _, e := range s.After {
				blit(effect.Frames[e.Frame], e.X)
				if e.Frame == 30 {
					blank++
				}
			}
			if !bytes.Equal(out, frame.Pixels) || s.Phase == "" {
				t.Fatalf("frame %d indexed sources differ", count)
			}
			if frame.HPStage != 0 {
				stages++
			}
			if frame.PlaySub1 {
				sub1++
			}
			if frame.PlaySub2 {
				sub2++
			}
			count++
		}
	}
	if count != 119 || stages != 20 || sub1 != 1 || sub2 != 1 || blank != 1 ||
		seq.SlideIn[10].Sources.ActorFrame != -1 || seq.SlideIn[9].Sources.ActorOffsetX != -90 ||
		seq.SlideOut[0].Sources.ActorOffsetX != -70 || seq.SlideOut[7].Sources.ActorOffsetX != 0 {
		t.Fatalf("sources changed phase/stage/sound/blank/slide: %d/%d/%d/%d/%d", count, stages, sub1, sub2, blank)
	}
}

func command9PresentAnimation(count int, delay int) *figani.Animation {
	frames := make([]figani.Frame, count)
	for index := range frames {
		frames[index] = figani.Frame{Width: 1, Height: 1, X: 1, Y: 1, Delay: delay, Pixels: []byte{byte(index + 1)}, Mask: []byte{1}}
	}
	return &figani.Animation{Frames: frames, HeaderByte2: byte(count)}
}

func TestBuildNativeCommand9AISequencePreservesAllPhases(t *testing.T) {
	effect := command9PresentAnimation(31, 1)
	schedule, err := figani.BuildNativeCommand9AISchedule(0, effect)
	if err != nil {
		t.Fatal(err)
	}
	bases := make([][]byte, figani.NativeCommand9AIDamageStages+1)
	for index := range bases {
		bases[index] = make([]byte, 320*200)
	}
	seq, err := BuildNativeCommand9AISequence(NativeCommand9AIInput{BeforeBase: bases[0], AfterBase: bases[len(bases)-1], TargetBases: bases, Actor: command9PresentAnimation(1, 1), Target: command9PresentAnimation(2, 1), Effect: effect, Schedule: schedule})
	if err != nil {
		t.Fatal(err)
	}
	if len(seq.SlideIn) != 11 || len(seq.Front) != 20 || len(seq.Target) != 60 || len(seq.Tail) != 20 || len(seq.SlideOut) != 8 {
		t.Fatalf("sequence lengths=%d/%d/%d/%d/%d", len(seq.SlideIn), len(seq.Front), len(seq.Target), len(seq.Tail), len(seq.SlideOut))
	}
	stages := 0
	for _, frame := range seq.Target {
		if frame.HPStage != 0 {
			stages++
		}
	}
	if stages != 20 {
		t.Fatalf("HP stages=%d", stages)
	}
}

func TestBuildNativeCommand9AISequenceRejectsMissingStage(t *testing.T) {
	effect := command9PresentAnimation(31, 1)
	schedule, _ := figani.BuildNativeCommand9AISchedule(0, effect)
	if _, err := BuildNativeCommand9AISequence(NativeCommand9AIInput{BeforeBase: make([]byte, 320*200), AfterBase: make([]byte, 320*200), TargetBases: make([][]byte, 20), Actor: command9PresentAnimation(1, 1), Target: command9PresentAnimation(1, 1), Effect: effect, Schedule: schedule}); err == nil {
		t.Fatal("missing HP stage was accepted")
	}
}
