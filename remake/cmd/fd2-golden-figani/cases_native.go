package main

import (
	"errors"
	"math"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// ---- resolver 規格 ----

// resolverSpec 第 i 次呼叫用 spec[i%len]：[擲骰步數, 是否命中, 是否回傳錯誤]。
type resolverSpec [][3]int

func (e *exporter) randomSpec() resolverSpec {
	n := 1 + e.rng.IntN(4)
	spec := make(resolverSpec, n)
	for i := range spec {
		errFlag := 0
		if e.rng.IntN(9) == 0 {
			errFlag = 1
		}
		spec[i] = [3]int{e.rng.IntN(4), e.rng.IntN(3) % 2, errFlag}
		if e.rng.IntN(3) == 0 {
			spec[i][1] = 1
		}
	}
	return spec
}

type walkFunc func(rng uint16, resolve func(int, uint16) (uint16, bool, error)) (uint16, error)

func (e *exporter) walk(spec resolverSpec, isNil bool, rng uint16, run walkFunc) obj {
	calls := [][]int{}
	resolve := func(index int, rng uint16) (uint16, bool, error) {
		calls = append(calls, ints(index, int(rng)))
		s := spec[index%len(spec)]
		if s[2] != 0 {
			return rng ^ 0x5555, true, errors.New("resolver failed")
		}
		next := rng
		for i := 0; i < s[0]; i++ {
			next = fdother.NativeRNGStep(next)
		}
		return next, s[1] != 0, nil
	}
	if isNil {
		resolve = nil
	}
	final, err := run(rng, resolve)
	var errJSON any
	if err != nil {
		errJSON = e.errText(err)
	}
	return obj{"rng": int(final), "err": errJSON, "calls": calls}
}

func specJSON(spec resolverSpec) [][]int {
	out := make([][]int, len(spec))
	for i, s := range spec {
		// go.mod 是 go 1.21，迴圈變數共用；必須複製，不能取 s[:]。
		out[i] = []int{s[0], s[1], s[2]}
	}
	return out
}

func (e *exporter) walkInput(spec resolverSpec, isNil bool, rng uint16, targets int) obj {
	return obj{"spec": specJSON(spec), "nil": isNil, "rng": int(rng), "targets": targets}
}

func (e *exporter) randomRNG() uint16 { return uint16(e.rng.Uint32()) }

func bankAnimation(count int, delay int) *figani.Animation {
	frames := make([]figani.Frame, count)
	for i := range frames {
		frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(i)}, Mask: []byte{1}, Delay: delay}
	}
	return &figani.Animation{Frames: frames, HeaderByte2: byte(count)}
}

func pickInt(e *exporter, lo, hi int) int { return lo + e.rng.IntN(hi-lo+1) }

func (e *exporter) seqResult(frames any, state any, err error) obj {
	if err != nil {
		return obj{"err": e.errText(err), "state": state}
	}
	return obj{"frames": frames, "state": state}
}

func (e *exporter) nativeCases() {
	e.command0Cases()
	e.command1Cases()
	e.command2Cases()
	e.command3Cases()
	e.command5Cases()
	e.command6Cases()
	e.command7Cases()
	e.command8Cases()
	e.command9Cases()
	e.compoundCases()
	e.add("builders_nil", "builders_nil", obj{}, e.builders(nil))
}

func (e *exporter) command0Cases() {
	grid := [][]any{}
	for step := -2; step <= 29; step++ {
		for layer := -2; layer <= 8; layer++ {
			frame, ok := figani.NativeCommand0EffectFrame(step, layer)
			grid = append(grid, []any{frame, ok, figani.NativeCommand0Impact(step, layer)})
		}
	}
	markers := []bool{}
	for step := -2; step <= 29; step++ {
		markers = append(markers, figani.NativeCommand0MarkerStep(step))
	}
	e.add("c0", "c0_grid", obj{}, obj{"grid": grid, "markers": markers})
	for i := 0; i < 60; i++ {
		frames := []int{28, 28, 28, 27, 29, 0}[e.rng.IntN(6)]
		spec, isNil, rng, targets := e.randomSpec(), i%13 == 5, e.randomRNG(), pickInt(e, -1, 4)
		input := e.walkInput(spec, isNil, rng, targets)
		input["frames"] = frames
		schedule := figani.NativeCommand0PresentationSchedule{Frames: frames}
		e.add("c0_walk", "c0_walk", input, e.walk(spec, isNil, rng, func(rng uint16, resolve func(int, uint16) (uint16, bool, error)) (uint16, error) {
			return figani.WalkNativeCommand0RNG(rng, schedule, targets, resolve)
		}))
	}
}

func (e *exporter) command1Cases() {
	counters := []any{}
	markers := []any{}
	for step := -2; step <= 32; step++ {
		c, err := figani.NativeCommand1Counters(step)
		if err != nil {
			counters = append(counters, e.errObj(err))
		} else {
			counters = append(counters, c[:])
		}
		numeric, sample, err := figani.NativeCommand1TargetMarkers(step)
		if err != nil {
			markers = append(markers, e.errObj(err))
		} else {
			markers = append(markers, []bool{numeric, sample})
		}
	}
	modes := [][]any{}
	for mode := 2; mode <= 6; mode++ {
		for slot := -2; slot <= 9; slot++ {
			for counter := -2; counter <= 16; counter++ {
				frame, offset, ok := figani.NativeCommand1ModeFrame(mode, slot, counter)
				modes = append(modes, []any{frame, offset, ok})
			}
		}
	}
	e.add("c1", "c1_grid", obj{}, obj{"counters": counters, "markers": markers, "modes": modes})
}

func (e *exporter) randomBank(maxFrames int) (obj, *figani.Animation) {
	r := e.rng
	count := r.IntN(maxFrames + 1)
	delays := make([]int, count)
	uniform := r.IntN(3) == 0
	for i := range delays {
		if uniform {
			delays[i] = 4
		} else {
			delays[i] = r.IntN(6) - 1
		}
	}
	h1, h2, h4 := 0, count, 0
	if r.IntN(6) == 0 {
		h1 = 1
	}
	if r.IntN(6) == 0 {
		h2 = r.IntN(40)
	}
	if r.IntN(8) == 0 {
		h4 = 1
	}
	a := &figani.Animation{Frames: make([]figani.Frame, count), HeaderByte1: byte(h1), HeaderByte2: byte(h2), HeaderByte4: byte(h4)}
	for i := range a.Frames {
		a.Frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{byte(i)}, Mask: []byte{1}, Delay: delays[i]}
	}
	return obj{"delays": delays, "h1": h1, "h2": h2, "h4": h4}, a
}

func (e *exporter) command2Cases() {
	target, end, err := figani.BuildNativeCommand2TargetSequence()
	e.add("c2", "c2_target", obj{}, e.seqResult(c2FramesJSON(target), c2StateJSON(end), err))
	for i := 0; i < 80; i++ {
		bank, a := e.randomBank(20)
		isNil := i%15 == 4
		if isNil {
			a = nil
		}
		side, repeat := byte(e.rng.IntN(3)), byte(e.rng.IntN(5))
		state := figani.NativeCommand2HelperState{Frame: byte(e.rng.Uint32()), Repeat: byte(e.rng.Uint32())}
		if e.rng.IntN(2) == 0 {
			state.Frame = byte(14 + e.rng.IntN(6))
		}
		front, frontEnd, err := figani.BuildNativeCommand2FrontSequence(side, a)
		frontJSON := e.seqResult(c2FramesJSON(front), c2StateJSON(frontEnd), err)
		tail, err := figani.BuildNativeCommand2TailSequence(side, repeat, a)
		var tailJSON any = c2FramesJSON(tail)
		if err != nil {
			tailJSON = e.errObj(err)
		}
		schedule, err := figani.BuildNativeCommand2PresentationSchedule(side, a)
		var scheduleJSON any = c2JSON(schedule)
		if err != nil {
			scheduleJSON = e.errObj(err)
		}
		input := obj{"bank": bank, "nil": isNil, "side": int(side), "repeat": int(repeat), "state": c2StateJSON(state)}
		e.add("c2_seq", "c2_seq", input, obj{
			"front": frontJSON, "tail": tailJSON, "schedule": scheduleJSON,
			"transition": c2FramesJSON(figani.BuildNativeCommand2TransitionSequence(state)),
		})
	}
	for i := 0; i < 40; i++ {
		spec, isNil, rng, targets := e.randomSpec(), i%11 == 3, e.randomRNG(), pickInt(e, -1, 4)
		e.add("c2_walk", "c2_walk", e.walkInput(spec, isNil, rng, targets), e.walk(spec, isNil, rng, func(rng uint16, resolve func(int, uint16) (uint16, bool, error)) (uint16, error) {
			return figani.WalkNativeCommand2RNG(rng, targets, resolve)
		}))
	}
}

// ---- 指令 3 ----

func (e *exporter) command3Cases() {
	r := e.rng
	for i := 0; i < 100; i++ {
		side := byte(r.IntN(3))
		schedule, _ := figani.BuildNativeCommand3PresentationSchedule(byte(r.IntN(2)), bankAnimation(33, 0))
		if r.IntN(6) == 0 {
			schedule.EffectResource = []int{0, 7}[r.IntN(2)]
		}
		if r.IntN(8) == 0 {
			schedule.SoundResource = []int{0, 85}[r.IntN(2)]
		}
		state := figani.NewNativeCommand3State()
		if i%4 != 0 {
			for c := range state.Counters {
				state.Counters[c] = r.IntN(40) - 25
				state.Positions[c] = r.IntN(12)
				if r.IntN(40) == 0 {
					state.Positions[c] = []int{-1, 12}[r.IntN(2)]
				}
			}
			state.NextPosition = r.IntN(18) - 3
			state.Stop = r.IntN(4) == 0
			state.Toggle = r.IntN(6) - 2
		}
		plan, err := figani.PlanNativeCommand3DrawFrame(state, schedule, side)
		var planJSON any = c3FrameJSON(plan)
		if err != nil {
			planJSON = e.errObj(err)
		}
		frames := func(list []figani.NativeCommand3Frame) []obj {
			out := make([]obj, len(list))
			for i, f := range list {
				out[i] = c3FrameJSON(f)
			}
			return out
		}
		target, afterTarget, err := figani.BuildNativeCommand3TargetSequence(state, schedule, side)
		targetJSON := e.seqResult(frames(target), c3StateJSON(afterTarget), err)
		transition, afterTransition, err := figani.BuildNativeCommand3TransitionSequence(state, schedule, side)
		transitionFrames := make([]obj, len(transition))
		for i, f := range transition {
			j := c3FrameJSON(f.NativeCommand3Frame)
			j["use_next"], j["off"] = f.UseNextTarget, f.TargetOffsetX
			transitionFrames[i] = j
		}
		transitionJSON := e.seqResult(transitionFrames, c3StateJSON(afterTransition), err)
		tail, afterTail, err := figani.BuildNativeCommand3TailSequence(state, schedule, side)
		tailJSON := e.seqResult(frames(tail), c3StateJSON(afterTail), err)
		input := obj{"schedule": c3SchedJSON(schedule), "state": c3StateJSON(state), "side": int(side)}
		e.add("c3_plan", "c3_plan", input, obj{"plan": planJSON, "target": targetJSON, "transition": transitionJSON, "tail": tailJSON})
	}
}

// ---- 指令 4／5 ----

func (e *exporter) randomC5Schedule() figani.NativeCommand5PresentationSchedule {
	r := e.rng
	side := byte(r.IntN(2))
	var s figani.NativeCommand5PresentationSchedule
	if r.IntN(2) == 0 {
		s, _ = figani.BuildNativeCommand4PresentationSchedule(side, bankAnimation(14, 0))
	} else {
		s, _ = figani.BuildNativeCommand5PresentationSchedule(side, bankAnimation(12, 0))
	}
	for k := 0; k < r.IntN(3); k++ {
		switch r.IntN(10) {
		case 0:
			s.CommandID = pickInt(e, 3, 6)
		case 1:
			s.EffectResource = []int{0, 22, 23, 24, 25}[r.IntN(5)]
		case 2:
			s.PhaseStride = r.IntN(9)
		case 3:
			s.ActiveCounters = r.IntN(9)
		case 4:
			s.EffectFrames = r.IntN(17)
		case 5:
			s.MarkerCounter = r.IntN(10)
		case 6:
			s.ResetCounter = r.IntN(11)
		case 7:
			s.TargetFrames = r.IntN(15)
		case 8:
			s.TailFrames = r.IntN(15)
		case 9:
			s.SampleEveryChannel = !s.SampleEveryChannel
		}
	}
	return s
}

func (e *exporter) c5Frames(list []figani.NativeCommand5Frame) []obj {
	out := make([]obj, len(list))
	for i, f := range list {
		out[i] = c5FrameJSON(f)
	}
	return out
}

func (e *exporter) command5Cases() {
	r := e.rng
	for i := 0; i < 120; i++ {
		schedule := e.randomC5Schedule()
		side := byte(r.IntN(3))
		rng := e.randomRNG()
		state := figani.NewNativeCommand5StateForSchedule(rng, schedule)
		if i%3 != 0 {
			for c := range state.Counters {
				state.Counters[c] = r.IntN(22) - 12
				state.Positions[c] = r.IntN(10)
				if r.IntN(30) == 0 {
					state.Positions[c] = []int{-1, 10}[r.IntN(2)]
				}
				state.Phases[c] = r.IntN(15)
			}
			state.NextPosition = r.IntN(15) - 3
			state.Stop = r.IntN(4) == 0
			state.RNG = e.randomRNG()
		}
		plan, err := figani.PlanNativeCommand5DrawFrame(state, schedule, side)
		var planJSON, shakeJSON any = c5FrameJSON(plan), c5FrameJSON(figani.NativeCommand5MarkerShake(plan))
		if err != nil {
			planJSON, shakeJSON = e.errObj(err), nil
		}
		hit, afterHit, err := figani.BuildNativeCommand5TargetSequenceHit(state, schedule, side, true)
		hitJSON := e.seqResult(e.c5Frames(hit), c5StateJSON(afterHit), err)
		miss, afterMiss, err := figani.BuildNativeCommand5TargetSequenceHit(state, schedule, side, false)
		missJSON := e.seqResult(e.c5Frames(miss), c5StateJSON(afterMiss), err)
		target, afterTarget, err := figani.BuildNativeCommand5TargetSequence(state, schedule, side)
		targetJSON := e.seqResult(e.c5Frames(target), c5StateJSON(afterTarget), err)
		transition, afterTransition, err := figani.BuildNativeCommand5TransitionSequence(state, schedule, side)
		transitionFrames := make([]obj, len(transition))
		for i, f := range transition {
			j := c5FrameJSON(f.NativeCommand5Frame)
			j["use_next"], j["off"] = f.UseNextTarget, f.TargetOffsetX
			transitionFrames[i] = j
		}
		transitionJSON := e.seqResult(transitionFrames, c5StateJSON(afterTransition), err)
		tail, afterTail, err := figani.BuildNativeCommand5TailSequence(state, schedule, side)
		tailJSON := e.seqResult(e.c5Frames(tail), c5StateJSON(afterTail), err)
		input := obj{"schedule": c5SchedJSON(schedule), "state": c5StateJSON(state), "side": int(side), "rng": int(rng)}
		e.add("c5_plan", "c5_plan", input, obj{
			"new_for_schedule": c5StateJSON(figani.NewNativeCommand5StateForSchedule(rng, schedule)),
			"new":              c5StateJSON(figani.NewNativeCommand5State(rng)),
			"plan":             planJSON, "shake": shakeJSON, "hit": hitJSON, "miss": missJSON, "target": targetJSON,
			"transition": transitionJSON, "tail": tailJSON,
		})
	}
	for i := 0; i < 60; i++ {
		schedule := e.randomC5Schedule()
		if i%4 == 0 {
			schedule, _ = figani.BuildNativeCommand4PresentationSchedule(byte(i%8/4), bankAnimation(14, 0))
		}
		side := byte(r.IntN(3))
		spec, isNil, rng, targets := e.randomSpec(), i%13 == 6, e.randomRNG(), pickInt(e, -1, 4)
		input := e.walkInput(spec, isNil, rng, targets)
		input["schedule"], input["side"] = c5SchedJSON(schedule), int(side)
		e.add("c5_walk", "c5_walk", input, e.walk(spec, isNil, rng, func(rng uint16, resolve func(int, uint16) (uint16, bool, error)) (uint16, error) {
			return figani.WalkNativeCommand5RNG(rng, schedule, side, targets, resolve)
		}))
	}
}

// ---- 指令 6 ----

func (e *exporter) randomC6Schedule() figani.NativeCommand6PresentationSchedule {
	s, _ := figani.BuildNativeCommand6PresentationSchedule(byte(e.rng.IntN(3)), bankAnimation(10, 0))
	if e.rng.IntN(7) == 0 {
		s.EffectResource = []int{0, 31, 34}[e.rng.IntN(3)]
	}
	if e.rng.IntN(7) == 0 {
		s.BaseByte = byte(e.rng.Uint32())
	}
	return s
}

func (e *exporter) randomPoints() [5]figani.NativeCommand6Point {
	if e.rng.IntN(2) == 0 {
		return figani.NativeCommand6Coordinates(pickInt(e, -60, 130), byte(e.rng.Uint32()))
	}
	var points [5]figani.NativeCommand6Point
	for i := range points {
		points[i] = figani.NativeCommand6Point{X: pickInt(e, -200, 400), Y: pickInt(e, -200, 300)}
	}
	return points
}

func (e *exporter) randomC6State() figani.NativeCommand6TargetState {
	state := figani.NewNativeCommand6TargetState()
	if e.rng.IntN(4) == 0 {
		return state
	}
	for c := range state.Counters {
		// 5..9 會讓 Go 以 frame 索引五筆表格而 panic，黃金資料不納入。
		state.Counters[c] = pickInt(e, -6, 4)
		if e.rng.IntN(30) == 0 {
			state.Counters[c] = pickInt(e, 10, 12)
		}
		state.Secondary[c] = byte(e.rng.IntN(10))
		if e.rng.IntN(30) == 0 {
			state.Secondary[c] = byte(pickInt(e, 10, 11))
		}
	}
	return state
}

func (e *exporter) command6Cases() {
	r := e.rng
	coords := [][]any{}
	radii := []int{1 << 40, -(1 << 40), 1 << 53, 1 << 62, math.MaxInt, math.MinInt, math.MaxInt / 2}
	for radius := -60; radius <= 130; radius++ {
		radii = append(radii, radius)
	}
	for _, radius := range radii {
		for _, base := range []byte{0, 30, 90, 255} {
			coords = append(coords, []any{radius, int(base), c6PointsJSON(figani.NativeCommand6Coordinates(radius, base))})
		}
	}
	e.add("c6", "c6_coords", obj{}, coords)
	for i := 0; i < 40; i++ {
		points, next := e.randomPoints(), pickInt(e, -50, 80)
		e.add("c6_target_coords", "c6_target_coords", obj{"points": c6PointsJSON(points), "next": next},
			c6PointsJSON(figani.NativeCommand6TargetCoordinates(points, next)))
	}
	for i := 0; i < 60; i++ {
		schedule, radius, side := e.randomC6Schedule(), pickInt(e, -40, 90), byte(r.IntN(3))
		prelude, err := figani.PlanNativeCommand6PreludeFrame(radius, side, schedule)
		var preludeJSON any = c6OrbitJSON(prelude)
		if err != nil {
			preludeJSON = e.errObj(err)
		}
		tail, err := figani.PlanNativeCommand6TailFrame(radius, side, schedule)
		var tailJSON any = c6OrbitJSON(tail)
		if err != nil {
			tailJSON = e.errObj(err)
		}
		e.add("c6_orbit", "c6_orbit", obj{"schedule": c6SchedJSON(schedule), "radius": radius, "side": int(side)},
			obj{"prelude": preludeJSON, "tail": tailJSON})
	}
	for i := 0; i < 120; i++ {
		schedule, points, state, side := e.randomC6Schedule(), e.randomPoints(), e.randomC6State(), byte(r.IntN(3))
		plan, err := figani.PlanNativeCommand6TargetFrame(state, schedule, points, side)
		var planJSON any = c6FrameJSON(plan)
		if err != nil {
			planJSON = e.errObj(err)
		}
		sequence, err := figani.BuildNativeCommand6TargetSequenceFromState(state, schedule, points, side)
		var sequenceJSON any
		if err != nil {
			sequenceJSON = e.errObj(err)
		} else {
			list := make([]obj, len(sequence))
			for i, f := range sequence {
				list[i] = c6FrameJSON(f)
			}
			sequenceJSON = list
		}
		transition, err := figani.BuildNativeCommand6TransitionSequence(state, schedule, points, side)
		var transitionJSON any
		if err != nil {
			transitionJSON = e.errObj(err)
		} else {
			list := make([]obj, len(transition))
			for i, f := range transition {
				list[i] = obj{"m4": c6LayersJSON(f.Mode4), "m5": c6LayersJSON(f.Mode5), "use_next": f.UseNextTarget, "off": f.TargetOffsetX, "next": c6StateJSON(f.Next)}
			}
			transitionJSON = list
		}
		input := obj{"schedule": c6SchedJSON(schedule), "points": c6PointsJSON(points), "state": c6StateJSON(state), "side": int(side)}
		e.add("c6_plan", "c6_plan", input, obj{"plan": planJSON, "sequence": sequenceJSON, "transition": transitionJSON})
	}
	for i := 0; i < 40; i++ {
		schedule, side := e.randomC6Schedule(), byte(r.IntN(3))
		spec, isNil, rng, targets := e.randomSpec(), i%11 == 2, e.randomRNG(), pickInt(e, -1, 4)
		input := e.walkInput(spec, isNil, rng, targets)
		input["schedule"], input["side"] = c6SchedJSON(schedule), int(side)
		e.add("c6_walk", "c6_walk", input, e.walk(spec, isNil, rng, func(rng uint16, resolve func(int, uint16) (uint16, bool, error)) (uint16, error) {
			return figani.WalkNativeCommand6RNG(rng, schedule, side, targets, resolve)
		}))
	}
	for i := 0; i < 60; i++ {
		side := byte(1 + r.IntN(2))
		schedule, _ := figani.BuildNativeCommand6PresentationSchedule(side, bankAnimation(10, 0))
		points := e.randomPoints()
		planned, _ := figani.BuildNativeCommand6TargetSequence(schedule, points, side)
		keep := len(planned)
		if r.IntN(8) == 0 {
			keep = r.IntN(len(planned) + 1)
		}
		state := figani.NewNativeCommand6DisplayState()
		if i%3 != 0 {
			state = figani.NativeCommand6DisplayState{Shade: pickInt(e, 1, 9), Pose: pickInt(e, -1, 4), Jitter: pickInt(e, -2, 2)}
		}
		hit, rng := r.IntN(4) != 0, e.randomRNG()
		draws, err := figani.BuildNativeCommand6TargetDisplayFrames(state, planned[:keep], hit, rng)
		var result any = c6DisplayJSON(draws)
		if err != nil {
			result = e.errObj(err)
		}
		input := obj{"side": int(side), "points": c6PointsJSON(points), "keep": keep,
			"state": ints(state.Shade, state.Pose, state.Jitter), "hit": hit, "rng": int(rng)}
		e.add("c6_display", "c6_display", input, result)
	}
}

// ---- 指令 7 ----

func (e *exporter) command7Cases() {
	r := e.rng
	c7Frames := func(list []figani.NativeCommand7Frame) []obj {
		out := make([]obj, len(list))
		for i, f := range list {
			out[i] = c7FrameJSON(f)
		}
		return out
	}
	randomSchedule := func() figani.NativeCommand7PresentationSchedule {
		s, _ := figani.BuildNativeCommand7PresentationSchedule(byte(r.IntN(2)), bankAnimation(5, 2))
		if r.IntN(7) == 0 {
			s.EffectResource = []int{0, 36}[r.IntN(2)]
		}
		return s
	}
	for i := 0; i < 100; i++ {
		schedule, side := randomSchedule(), byte(r.IntN(3))
		state := figani.NewNativeCommand7State()
		if i%4 != 0 {
			for c := range state.Counters {
				state.Counters[c] = r.IntN(22) - 12
				state.Positions[c] = r.IntN(10)
				if r.IntN(30) == 0 {
					state.Positions[c] = []int{-1, 10}[r.IntN(2)]
				}
			}
			state.NextPosition = r.IntN(15) - 3
			state.Stop = r.IntN(4) == 0
			state.Toggle = byte(r.Uint32())
		}
		plan, err := figani.PlanNativeCommand7DrawFrame(state, schedule, side)
		var planJSON any = c7FrameJSON(plan)
		if err != nil {
			planJSON = e.errObj(err)
		}
		target, afterTarget, err := figani.BuildNativeCommand7TargetSequence(state, schedule, side)
		targetJSON := e.seqResult(c7Frames(target), c7StateJSON(afterTarget), err)
		transition, afterTransition, err := figani.BuildNativeCommand7TransitionSequence(state, schedule, side)
		transitionFrames := make([]obj, len(transition))
		for i, f := range transition {
			j := c7FrameJSON(f.NativeCommand7Frame)
			j["use_next"], j["off"] = f.UseNextTarget, f.TargetOffsetX
			transitionFrames[i] = j
		}
		transitionJSON := e.seqResult(transitionFrames, c7StateJSON(afterTransition), err)
		tail, afterTail, err := figani.BuildNativeCommand7TailSequence(state, schedule, side)
		tailJSON := e.seqResult(c7Frames(tail), c7StateJSON(afterTail), err)
		input := obj{"schedule": c7SchedJSON(schedule), "state": c7StateJSON(state), "side": int(side)}
		e.add("c7_plan", "c7_plan", input, obj{"plan": planJSON, "target": targetJSON, "transition": transitionJSON, "tail": tailJSON})
	}
	for i := 0; i < 50; i++ {
		schedule, side := randomSchedule(), byte(r.IntN(3))
		spec, isNil, rng, targets := e.randomSpec(), i%11 == 7, e.randomRNG(), pickInt(e, -1, 4)
		input := e.walkInput(spec, isNil, rng, targets)
		input["schedule"], input["side"] = c7SchedJSON(schedule), int(side)
		e.add("c7_walk", "c7_walk", input, e.walk(spec, isNil, rng, func(rng uint16, resolve func(int, uint16) (uint16, bool, error)) (uint16, error) {
			return figani.WalkNativeCommand7RNG(rng, schedule, side, targets, resolve)
		}))
	}
}

// ---- 指令 8 ----

func (e *exporter) command8Cases() {
	r := e.rng
	c8Frames := func(list []figani.NativeCommand8Frame) []obj {
		out := make([]obj, len(list))
		for i, f := range list {
			out[i] = c8FrameJSON(f)
		}
		return out
	}
	for i := 0; i < 80; i++ {
		schedule, _ := figani.BuildNativeCommand8PresentationSchedule(byte(r.IntN(2)), bankAnimation(32, 0))
		if r.IntN(7) == 0 {
			schedule.EffectResource = 0
		}
		if r.IntN(7) == 0 {
			schedule.SoundResource = []int{0, 89}[r.IntN(2)]
		}
		if r.IntN(6) == 0 {
			schedule.FrameBases[r.IntN(16)] = pickInt(e, -3, 30)
		}
		side := byte(r.IntN(3))
		state := figani.NewNativeCommand8State()
		if i%4 != 0 {
			for c := range state.Counters {
				state.Counters[c] = r.IntN(46) - 35
			}
		}
		plan, err := figani.PlanNativeCommand8DrawFrame(state, schedule)
		var planJSON any = c8FrameJSON(plan)
		if err != nil {
			planJSON = e.errObj(err)
		}
		target, afterTarget, err := figani.BuildNativeCommand8TargetSequence(state, schedule)
		targetJSON := e.seqResult(c8Frames(target), afterTarget.Counters[:], err)
		transition, afterTransition, err := figani.BuildNativeCommand8TransitionSequence(state, schedule, side)
		transitionFrames := make([]obj, len(transition))
		for i, f := range transition {
			j := c8FrameJSON(f.NativeCommand8Frame)
			j["use_next"], j["off"] = f.UseNextTarget, f.TargetOffsetX
			transitionFrames[i] = j
		}
		transitionJSON := e.seqResult(transitionFrames, afterTransition.Counters[:], err)
		tail, afterTail, err := figani.BuildNativeCommand8TailSequence(state, schedule)
		tailJSON := e.seqResult(c8Frames(tail), afterTail.Counters[:], err)
		input := obj{"schedule": c8SchedJSON(schedule), "state": state.Counters[:], "side": int(side)}
		e.add("c8_plan", "c8_plan", input, obj{"plan": planJSON, "target": targetJSON, "transition": transitionJSON, "tail": tailJSON})
	}
}

// ---- 指令 9 ----

func (e *exporter) command9Cases() {
	r := e.rng
	for i := 0; i < 80; i++ {
		count := 31
		if r.IntN(5) == 0 {
			count = pickInt(e, 28, 33)
		}
		side := byte(0)
		if r.IntN(5) == 0 {
			side = byte(pickInt(e, 1, 2))
		}
		schedule, err := figani.BuildNativeCommand9AISchedule(side, bankAnimation(count, 0))
		var scheduleJSON any = c9SchedJSON(schedule)
		if err != nil {
			scheduleJSON = e.errObj(err)
			schedule, _ = figani.BuildNativeCommand9AISchedule(0, bankAnimation(31, 0))
		}
		if r.IntN(6) == 0 {
			schedule.EffectResource = []int{0, 43}[r.IntN(2)]
		}
		if r.IntN(5) == 0 {
			schedule.TargetFrames = pickInt(e, 0, 62)
		}
		if r.IntN(5) == 0 {
			schedule.DamageStages = pickInt(e, 0, 30)
		}
		state := figani.NewNativeCommand9AIState()
		if i%3 != 0 {
			state = figani.NativeCommand9AIState{Counter: pickInt(e, -1, 62), Toggle: r.IntN(2) == 0}
		}
		orbit, err := figani.PlanNativeCommand9AIOrbitFrame(state, schedule)
		var orbitJSON any = c9FrameJSON(orbit)
		if err != nil {
			orbitJSON = e.errObj(err)
		}
		plan, err := figani.PlanNativeCommand9AITargetFrame(state, schedule)
		var planJSON any = c9FrameJSON(plan)
		if err != nil {
			planJSON = e.errObj(err)
		}
		sequence, after, err := figani.BuildNativeCommand9AITargetSequence(state, schedule)
		frames := make([]obj, len(sequence))
		for i, f := range sequence {
			frames[i] = c9FrameJSON(f)
		}
		sequenceJSON := e.seqResult(frames, []any{after.Counter, after.Toggle}, err)
		input := obj{"count": count, "side": int(side), "schedule": c9SchedJSON(schedule), "state": []any{state.Counter, state.Toggle}}
		e.add("c9_plan", "c9_plan", input, obj{"built": scheduleJSON, "orbit": orbitJSON, "plan": planJSON, "sequence": sequenceJSON})
	}
}

func (e *exporter) compoundCases() {
	grid := []any{}
	for id := 30; id <= 37; id++ {
		for _, count := range []int{-1, 0, 1, 2, 8} {
			var a *figani.Animation
			if count >= 0 {
				a = &figani.Animation{Frames: make([]figani.Frame, count)}
			}
			s, err := figani.BuildNativeCompoundPresentationSchedule(id, a)
			if err != nil {
				grid = append(grid, e.errObj(err))
			} else {
				grid = append(grid, compoundJSON(s))
			}
		}
	}
	e.add("compound", "compound_grid", obj{}, grid)
}
