package main

import (
	"bytes"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// 這個檔的 JSON 形狀必須與 remake-rs/crates/fd2-formats/tests/golden_figani.rs 逐鍵相同。

type obj = map[string]any

func ints(values ...int) []int { return values }

func bytesJSON(b []byte) []int {
	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}
	return out
}

// errText 以 $ROOT 取代合成目錄，讓 Go 與 Rust 的暫存路徑可比。
func (e *exporter) errText(err error) string {
	s := err.Error()
	if e.work != "" {
		s = strings.ReplaceAll(s, e.work, "$ROOT")
	}
	if e.pack != "" {
		s = strings.ReplaceAll(s, e.pack, "$PACK")
	}
	return s
}

func (e *exporter) errObj(err error) obj { return obj{"err": e.errText(err)} }

func frameJSON(f figani.Frame) obj {
	return obj{
		"x": f.X, "y": f.Y, "w": f.Width, "h": f.Height, "delay": f.Delay,
		"r4": int(f.RawByte4), "r5": int(f.RawByte5), "r7": int(f.RawByte7),
		"pix": digest(f.Pixels), "mask": digest(f.Mask), "npix": len(f.Pixels), "nmask": len(f.Mask),
	}
}

func animJSON(a *figani.Animation) obj {
	frames := make([]obj, len(a.Frames))
	for i, f := range a.Frames {
		frames[i] = frameJSON(f)
	}
	return obj{"h1": int(a.HeaderByte1), "h2": int(a.HeaderByte2), "h4": int(a.HeaderByte4), "frames": frames}
}

func c0JSON(s figani.NativeCommand0PresentationSchedule) obj {
	return obj{"effect": s.EffectResource, "sound": s.SoundResource, "sound_index": s.SoundIndex, "frames": s.Frames,
		"stages": s.DamageStages, "shift": s.SideXShift, "flags": bytesJSON(s.LayerFlags[:]), "x": s.XOffsets[:], "y": s.YOffsets[:]}
}

func c1JSON(s figani.NativeCommand1PresentationSchedule) obj {
	return obj{"effect": s.EffectResource, "shift": s.SideXShift, "anchor": s.AnchorX, "x": s.XOffsets[:], "y": s.YOffsets[:]}
}

func c2JSON(s figani.NativeCommand2PresentationSchedule) obj {
	return obj{"effect": s.EffectResource, "sound": s.SoundResource, "indices": s.SoundIndices[:]}
}

func c2StateJSON(s figani.NativeCommand2HelperState) []int { return ints(int(s.Frame), int(s.Repeat)) }

func c2FramesJSON(frames []figani.NativeCommand2HelperFrame) [][]any {
	out := make([][]any, len(frames))
	for i, f := range frames {
		out[i] = []any{f.EffectFrame, int(f.Next.Frame), int(f.Next.Repeat), f.Sample1, f.Sample2, f.NumericMarker, f.HPStage}
	}
	return out
}

func c3SchedJSON(s figani.NativeCommand3PresentationSchedule) obj {
	return obj{"effect": s.EffectResource, "sound": s.SoundResource, "x": s.XOffsets[:], "y": s.YOffsets[:], "bases": s.FrameBases[:]}
}

func c3StateJSON(s figani.NativeCommand3State) obj {
	return obj{"c": s.Counters[:], "p": s.Positions[:], "np": s.NextPosition, "stop": s.Stop, "t": s.Toggle}
}

func c3FrameJSON(f figani.NativeCommand3Frame) obj {
	layers := make([][]int, len(f.Layers))
	for i, l := range f.Layers {
		layers[i] = ints(l.Channel, l.Frame, l.X, l.Y)
	}
	return obj{"layers": layers, "num": f.NumericMarker, "s1": f.PlaySub1, "s2": f.PlaySub2, "hp": f.HPStage, "next": c3StateJSON(f.Next)}
}

func c5SchedJSON(s figani.NativeCommand5PresentationSchedule) obj {
	return obj{"cmd": s.CommandID, "effect": s.EffectResource, "sound": s.SoundResource, "indices": s.SoundIndices[:],
		"x": s.XOffsets[:], "ef": s.EffectFrames, "ps": s.PhaseStride, "ac": s.ActiveCounters, "ff": s.FrontFrames,
		"tf": s.TargetFrames, "tl": s.TailFrames, "mc": s.MarkerCounter, "rc": s.ResetCounter, "every": s.SampleEveryChannel}
}

func c5StateJSON(s figani.NativeCommand5State) obj {
	return obj{"c": s.Counters[:], "p": s.Positions[:], "ph": s.Phases[:], "np": s.NextPosition, "stop": s.Stop, "rng": int(s.RNG)}
}

func c5FrameJSON(f figani.NativeCommand5Frame) obj {
	layers := make([][]int, len(f.Layers))
	for i, l := range f.Layers {
		layers[i] = ints(l.Channel, l.Frame, l.X)
	}
	return obj{"layers": layers, "next": c5StateJSON(f.Next), "num": f.NumericMarker, "hp": f.HPStage, "pri": f.PlayPrimary, "sec": f.PlaySecondary}
}

func c6SchedJSON(s figani.NativeCommand6PresentationSchedule) obj {
	return obj{"effect": s.EffectResource, "sound": s.SoundResource, "indices": s.SoundIndices[:], "base": int(s.BaseByte),
		"dw": s.DwordTable[:], "bt": bytesJSON(s.ByteTable[:])}
}

func c6PointsJSON(points [5]figani.NativeCommand6Point) [][]int {
	out := make([][]int, len(points))
	for i, p := range points {
		out[i] = ints(p.X, p.Y)
	}
	return out
}

func c6StateJSON(s figani.NativeCommand6TargetState) obj {
	return obj{"c": s.Counters[:], "s": bytesJSON(s.Secondary[:])}
}

func c6LayersJSON(layers []figani.NativeCommand6Layer) [][]any {
	out := make([][]any, len(layers))
	for i, l := range layers {
		out[i] = []any{l.Mode, l.Channel, l.Frame, l.X, l.Y, l.Secondary}
	}
	return out
}

func c6FrameJSON(f figani.NativeCommand6TargetFrame) obj {
	return obj{"m4": c6LayersJSON(f.Mode4), "m5": c6LayersJSON(f.Mode5), "next": c6StateJSON(f.Next), "num": f.NumericMarker, "hp": f.HPStage}
}

func c6OrbitJSON(f figani.NativeCommand6OrbitFrame) obj {
	return obj{"first": c6LayersJSON(f.First), "second": c6LayersJSON(f.Second), "nr": f.NextRadius, "dt": f.DrawTarget}
}

func c6DisplayJSON(frames []figani.NativeCommand6TargetDisplayFrame) [][]int {
	out := make([][]int, len(frames))
	for i, f := range frames {
		out[i] = ints(f.Shader, f.OffsetX, f.OffsetY, int(f.RNGAfter), f.Next.Shade, f.Next.Pose, f.Next.Jitter)
	}
	return out
}

func c7SchedJSON(s figani.NativeCommand7PresentationSchedule) obj {
	return obj{"effect": s.EffectResource, "sound": s.SoundResource, "indices": s.SoundIndices[:], "x": s.XOffsets[:]}
}

func c7StateJSON(s figani.NativeCommand7State) obj {
	return obj{"c": s.Counters[:], "p": s.Positions[:], "np": s.NextPosition, "stop": s.Stop, "t": int(s.Toggle)}
}

func c7FrameJSON(f figani.NativeCommand7Frame) obj {
	layers := make([][]int, len(f.Layers))
	for i, l := range f.Layers {
		layers[i] = ints(l.Channel, l.Frame, l.X)
	}
	return obj{"layers": layers, "next": c7StateJSON(f.Next), "num": f.NumericMarker, "hp": f.HPStage, "a": f.PlayHandleA, "b": f.PlayHandleB}
}

func c8SchedJSON(s figani.NativeCommand8PresentationSchedule) obj {
	return obj{"effect": s.EffectResource, "sound": s.SoundResource, "bases": s.FrameBases[:]}
}

func c8FrameJSON(f figani.NativeCommand8Frame) obj {
	layers := make([][]int, len(f.Layers))
	for i, l := range f.Layers {
		layers[i] = ints(l.Channel, l.Frame)
	}
	return obj{"layers": layers, "num": f.NumericMarker, "s1": f.PlaySub1, "s2": f.PlaySub2, "hp": f.HPStage, "next": f.Next.Counters[:]}
}

func c9SchedJSON(s figani.NativeCommand9AISchedule) obj {
	return obj{"effect": s.EffectResource, "sound": s.SoundResource, "ff": s.FrontFrames, "tf": s.TargetFrames, "tl": s.TailFrames,
		"ds": s.DamageStages, "in": s.ActorSlideIn, "out": s.ActorSlideOut, "hold": s.SlideInHoldMS}
}

func c9FrameJSON(f figani.NativeCommand9AIFrame) obj {
	effect := f.EffectFrames
	if effect == nil {
		effect = []int{}
	}
	return obj{"ef": effect, "num": f.NumericMarker, "hp": f.HPStage, "s1": f.PlaySub1, "s2": f.PlaySub2, "next": []any{f.Next.Counter, f.Next.Toggle}}
}

func compoundJSON(s figani.NativeCompoundPresentationSchedule) obj {
	return obj{"cmd": s.CommandID, "effect": s.EffectResource, "sound": s.SoundResource, "asf": s.ActorSlideFrames, "asx": s.ActorSlideStepX,
		"esf": s.EffectSlideFrames, "esx": s.EffectSlideStepX, "phold": s.PreludeHoldTicks, "mdel": s.MainDelayTicks,
		"ovl": s.OverlayFirstFrame, "mfc": s.MainFrameCount, "ms2": s.MixerSample2Frame, "s1": s.Sample1Frame,
		"tail": s.TailEnabled, "tf": s.TailFrames, "rgb": bytesJSON(s.PaletteRGB[:])}
}

func cuesJSON(cues []figani.NativeCommandFrameCue) [][]int {
	out := make([][]int, len(cues))
	for i, c := range cues {
		out[i] = ints(c.Frame, c.Sample)
	}
	return out
}

func orEmpty(v []int) []int {
	if v == nil {
		return []int{}
	}
	return v
}

func strikeJSON(s figani.NativeCommandDerivedStrikeSchedule) obj {
	return obj{"cmd": s.CommandID, "ts": s.TargetStart, "aif": s.ActorImpactFrame, "tif": s.TargetImpactFrame,
		"tifs": orEmpty(s.TargetImpactFrames), "den": s.DamageDenominator, "audio": s.AudioResource, "as": s.ActorSample,
		"acues": cuesJSON(s.ActorCues), "tcues": cuesJSON(s.TargetCues), "tsamp": s.TargetSample, "delays": orEmpty(s.FrameDelays),
		"utb": s.UsesTargetBase, "pm": int(s.PreludeMode), "ubg": s.UsesBGTransition, "shake": s.ShakeOffsets[:]}
}

func c24JSON(s figani.NativeCommand24Schedule) obj {
	return obj{"ts": s.TargetStart, "aif": s.ActorImpactFrame, "tif": s.TargetImpactFrame, "audio": s.AudioResource,
		"as": s.ActorSample, "tsamp": s.TargetSample, "shake": s.ShakeOffsets[:]}
}

// builders 對一份動畫跑所有指令排程建構器（各 raw side），讓簽章檢查在真實與隨機輸入上都被比對。
func (e *exporter) builders(a *figani.Animation) obj {
	sides := []byte{0, 1, 2}
	pick := func(f func(byte) any) []any {
		out := make([]any, len(sides))
		for i, side := range sides {
			out[i] = f(side)
		}
		return out
	}
	res := func(v obj, err error) any {
		if err != nil {
			return e.errObj(err)
		}
		return v
	}
	out := obj{}
	out["c0"] = pick(func(s byte) any {
		v, err := figani.BuildNativeCommand0PresentationSchedule(s, a)
		return res(c0JSON(v), err)
	})
	out["c1"] = pick(func(s byte) any {
		v, err := figani.BuildNativeCommand1PresentationSchedule(s, a)
		return res(c1JSON(v), err)
	})
	out["c2"] = pick(func(s byte) any {
		v, err := figani.BuildNativeCommand2PresentationSchedule(s, a)
		return res(c2JSON(v), err)
	})
	out["c2front"] = pick(func(s byte) any {
		frames, end, err := figani.BuildNativeCommand2FrontSequence(s, a)
		if err != nil {
			return e.errObj(err)
		}
		tail, err := figani.BuildNativeCommand2TailSequence(s, end.Repeat, a)
		tailJSON := any(nil)
		if err != nil {
			tailJSON = e.errObj(err)
		} else {
			tailJSON = c2FramesJSON(tail)
		}
		return obj{"frames": c2FramesJSON(frames), "state": c2StateJSON(end), "tail": tailJSON}
	})
	out["c3"] = pick(func(s byte) any {
		v, err := figani.BuildNativeCommand3PresentationSchedule(s, a)
		return res(c3SchedJSON(v), err)
	})
	out["c4"] = pick(func(s byte) any {
		v, err := figani.BuildNativeCommand4PresentationSchedule(s, a)
		return res(c5SchedJSON(v), err)
	})
	out["c5"] = pick(func(s byte) any {
		v, err := figani.BuildNativeCommand5PresentationSchedule(s, a)
		return res(c5SchedJSON(v), err)
	})
	out["c6"] = pick(func(s byte) any {
		v, err := figani.BuildNativeCommand6PresentationSchedule(s, a)
		return res(c6SchedJSON(v), err)
	})
	out["c7"] = pick(func(s byte) any {
		v, err := figani.BuildNativeCommand7PresentationSchedule(s, a)
		return res(c7SchedJSON(v), err)
	})
	out["c8"] = pick(func(s byte) any {
		v, err := figani.BuildNativeCommand8PresentationSchedule(s, a)
		return res(c8SchedJSON(v), err)
	})
	out["c9"] = pick(func(s byte) any {
		v, err := figani.BuildNativeCommand9AISchedule(s, a)
		return res(c9SchedJSON(v), err)
	})
	c24, err := figani.BuildNativeCommand24Schedule(a)
	out["c24"] = res(c24JSON(c24), err)
	strike := []any{}
	for _, id := range []int{23, 24, 27, 28, 29, 30, 31, 32} {
		v, err := figani.BuildNativeCommandDerivedStrikeSchedule(a, id)
		strike = append(strike, res(strikeJSON(v), err))
	}
	out["strike"] = strike
	compound := []any{}
	for id := 31; id <= 36; id++ {
		v, err := figani.BuildNativeCompoundPresentationSchedule(id, a)
		compound = append(compound, res(compoundJSON(v), err))
	}
	out["compound"] = compound
	return out
}

// schedulers 以動畫的 delay 跑原生排程與顯示排程。
func (e *exporter) schedulers(a *figani.Animation, ticks int, steps int) obj {
	native := figani.NativeScheduler{}
	trace := [][]any{}
	for i := 0; i < steps; i++ {
		index, rendered, err := native.Step(a, i != 0)
		if err != nil {
			trace = append(trace, []any{e.errText(err)})
			break
		}
		trace = append(trace, []any{index, rendered, native.FrameIndex, native.Subframe})
	}
	delays := make([]int, len(a.Frames))
	for i, f := range a.Frames {
		delays[i] = f.Delay
	}
	return obj{"native": trace, "display": e.displayJSON(delays, ticks, 400)}
}

func (e *exporter) displayJSON(delays []int, ticks, limit int) any {
	s, err := figani.NewDisplayScheduler(delays, ticks)
	if err != nil {
		return e.errObj(err)
	}
	trace := [][]any{}
	for i := 0; i < limit; i++ {
		frame, presented, done, err := s.Step()
		if err != nil {
			trace = append(trace, []any{e.errText(err)})
			break
		}
		trace = append(trace, []any{frame, presented, done})
		if done && !presented {
			break
		}
	}
	starts := []any{}
	for frame := -1; frame <= len(delays); frame++ {
		start, ok := s.FrameStart(frame)
		starts = append(starts, []any{start, ok})
	}
	return obj{"body": s.BodyTicks(), "current": s.CurrentFrame(), "trace": trace, "starts": starts}
}

func (e *exporter) blitResult(dst []byte, before []byte, err error) any {
	if err != nil {
		return obj{"err": e.errText(err), "kept": bytes.Equal(dst, before)}
	}
	return obj{"sha": digest(dst), "changed": !bytes.Equal(dst, before)}
}

// frameBlits 把幀畫到三種原生畫面上：320 跨距、640 跨距右半、400 跨距暫存區（含覆寫色）。
func (e *exporter) frameBlits(f figani.Frame) []any {
	blit := func(size int, draw func([]byte) error) any {
		dst := bytes.Repeat([]byte{9}, size)
		before := append([]byte(nil), dst...)
		return e.blitResult(dst, before, draw(dst))
	}
	return []any{
		blit(320*200, func(dst []byte) error { return f.BlitAt(dst, 320) }),
		blit(640*200, func(dst []byte) error { return f.BlitAtBase(dst, 640, 320) }),
		blit(400*240, func(dst []byte) error { return f.BlitTranslated(dst, 400, -40, 20, nil) }),
		blit(400*240, func(dst []byte) error { return f.BlitTranslated(dst, 400, -40, 20, ptr(33)) }),
	}
}

func ptr(b byte) *byte { return &b }

// animationCase 是解析成功後的完整衍生結果。
func (e *exporter) animationResult(a *figani.Animation, withBlits bool) obj {
	out := obj{"anim": animJSON(a), "builders": e.builders(a), "sched": e.schedulers(a, 1+len(a.Frames)%3, 60)}
	if withBlits {
		blits := make([]any, len(a.Frames))
		for i, f := range a.Frames {
			blits[i] = e.frameBlits(f)
		}
		out["blits"] = blits
	}
	return out
}
