package battlepresent

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func nativeCompoundTestFrame(value byte) figani.Frame {
	return figani.Frame{X: 0, Y: 0, Width: 1, Height: 1, Pixels: []byte{value}, Mask: []byte{1}, Delay: 1}
}

func TestBuildNativeCompoundCommonFramesPreservesLayerOrder(t *testing.T) {
	effect := &figani.Animation{Frames: []figani.Frame{nativeCompoundTestFrame(10), nativeCompoundTestFrame(11), nativeCompoundTestFrame(12)}}
	schedule, err := figani.BuildNativeCompoundPresentationSchedule(32, effect)
	if err != nil {
		t.Fatal(err)
	}
	base := make([]byte, 320*200)
	frames, err := BuildNativeCompoundCommonFrames(base, nativeCompoundTestFrame(7), effect, schedule)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames.ActorSlide) != 8 || frames.ActorSlide[0][0] != 7 || frames.ActorSlide[7][140] != 7 ||
		len(frames.EffectSlide) != 9 || frames.EffectSlide[0][240] != 10 || frames.EffectSlide[8][0] != 10 ||
		len(frames.Main) != 2 || frames.Main[0][0] != 11 || frames.Main[1][0] != 12 || len(frames.Tail) != 11 ||
		frames.Tail[0][0] != 11 || frames.Tail[1][0] != 12 || frames.Tail[10][0] != 11 {
		t.Fatalf("unexpected compound frame schedule")
	}
}

func TestBuildNativeCompoundCommonFramesOverlayIsCommandSpecific(t *testing.T) {
	second := nativeCompoundTestFrame(11)
	second.X = 1
	effect := &figani.Animation{Frames: []figani.Frame{nativeCompoundTestFrame(10), second}}
	base := bytes.Repeat([]byte{3}, 320*200)
	s32, _ := figani.BuildNativeCompoundPresentationSchedule(32, effect)
	s33, _ := figani.BuildNativeCompoundPresentationSchedule(33, effect)
	f32, err := BuildNativeCompoundCommonFrames(base, nativeCompoundTestFrame(7), effect, s32)
	if err != nil {
		t.Fatal(err)
	}
	f33, err := BuildNativeCompoundCommonFrames(base, nativeCompoundTestFrame(7), effect, s33)
	if err != nil {
		t.Fatal(err)
	}
	if f32.Main[0][0] != 3 || f33.Main[0][0] != 10 || f32.Main[0][1] != 11 || f33.Main[0][1] != 11 {
		t.Fatal("command-specific steady overlay corrupted main frame")
	}
}

func TestBuildNativeCompoundCommonFramesRejectsWithoutPartialOutput(t *testing.T) {
	effect := &figani.Animation{Frames: []figani.Frame{nativeCompoundTestFrame(1)}}
	schedule := figani.NativeCompoundPresentationSchedule{CommandID: 32, EffectResource: 65}
	if frames, err := BuildNativeCompoundCommonFrames(make([]byte, 320*200), nativeCompoundTestFrame(2), effect, schedule); err == nil || len(frames.ActorSlide) != 0 {
		t.Fatal("malformed compound effect was accepted")
	}
}

func TestBuildNativeCompoundActorFramesPreservesDelayAndLayerOrder(t *testing.T) {
	actor := nativeCompoundTestFrame(7)
	actor.Delay = 2
	target := nativeCompoundTestFrame(9)
	target.Delay = 1
	actorAnimation := &figani.Animation{HeaderByte4: 2, Frames: []figani.Frame{actor}}
	targetAnimation := &figani.Animation{Frames: []figani.Frame{target}}
	base := make([]byte, 320*200)
	right, err := BuildNativeCompoundActorFrames(base, actorAnimation, targetAnimation, 0)
	if err != nil {
		t.Fatal(err)
	}
	left, err := BuildNativeCompoundActorFrames(base, actorAnimation, targetAnimation, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(right) != 2 || right[0][0] != 7 || left[0][0] != 9 {
		t.Fatalf("compound actor right=%v left=%v", right[0][0], left[0][0])
	}
}

func TestBuildNativeCompoundActorFramesRejectsMPMarkerBranch(t *testing.T) {
	actor := &figani.Animation{HeaderByte4: 1, Frames: []figani.Frame{nativeCompoundTestFrame(1)}}
	target := &figani.Animation{Frames: []figani.Frame{nativeCompoundTestFrame(2)}}
	if frames, err := BuildNativeCompoundActorFrames(make([]byte, 320*200), actor, target, 0); err == nil || len(frames) != 0 {
		t.Fatal("compound actor accepted unimplemented MP marker branch")
	}
}
