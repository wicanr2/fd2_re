// fd2-golden-present-bp 輸出 internal/battlepresent 的黃金差分資料，供 Rust 改寫
// remake-rs/crates/fd2-campaign 的 battlepresent 模組逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 battlepresent 與 figani／fdother 的公開 API，不改任何遊戲行為。
//   - 真實輸入：分離素材包 animations/ 底下用到的 FIGANI 與 FDOTHER 動畫（各指令的 effect bank 與一組
//     依固定種子挑出的 FIGANI 角色動畫）。檔案清單排序後連同大小與 SHA-256 寫進 JSON，Rust 端讀同一批
//     檔案前先核對。
//   - 產生輸入：以固定種子的 PCG 挑選陣營、步數、目標數、動畫與幀；320×200 底圖、工作緩衝、合成的
//     FDOTHER RLE 幀與 LMI1 項目則由 JSON 記下的 64 位元種子經下方 noise／gen 產生（Rust 端以相同的
//     一行 LCG 重建，不需要 Go 的亂數）。
//
// 繪製結果記每張畫格的 SHA-256；排程與序列記每一步的結構（來源層、HP 段、音效旗標、下一個 idle 狀態）。
// 錯誤欄位是 Go err.Error() 原文。
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type obj = map[string]any

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// lcg 是兩端共用的一行產生器：x = x*6364136223846793005 + 1442695040888963407，取高 32 位。
type lcg struct{ x uint64 }

func (g *lcg) next() uint32 {
	g.x = g.x*6364136223846793005 + 1442695040888963407
	return uint32(g.x >> 32)
}

func noise(seed uint64, n int) []byte {
	g := lcg{seed}
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(g.next() >> 24)
	}
	return out
}

// rleFrame 產生 w×h 的 FDOTHER 四模式 RLE 幀：每列以長度 1..64 的單色連續段組成（opcode = count-1）。
func rleFrame(seed uint64, w, h int) fdother.Frame {
	g := lcg{seed}
	raw := make([]byte, 4)
	binary.LittleEndian.PutUint16(raw, uint16(w))
	binary.LittleEndian.PutUint16(raw[2:], uint16(h))
	for y := 0; y < h; y++ {
		for remaining := w; remaining > 0; {
			count := 1 + int(g.next()%64)
			if count > remaining {
				count = remaining
			}
			raw = append(raw, byte(count-1), byte(g.next()))
			remaining -= count
		}
	}
	return fdother.Frame{Width: w, Height: h, Pixels: raw}
}

// lmiList 產生 count 個 1..3×1..3 的 LMI1 項目，像素值可為 0（透明）。
func lmiList(seed uint64, count int) []fdother.LMI1Entry {
	g := lcg{seed}
	out := make([]fdother.LMI1Entry, count)
	for i := range out {
		w, h := 1+int(g.next()%3), 1+int(g.next()%3)
		pixels := make([]byte, w*h)
		for p := range pixels {
			pixels[p] = byte(g.next())
		}
		out[i] = fdother.LMI1Entry{Width: w, Height: h, Pixels: pixels}
	}
	return out
}

type animRef struct {
	Src string
	ID  int
}

type exporter struct {
	rng    *rand.Rand
	pack   string
	cases  []obj
	files  []obj
	dirs   map[string]bool
	anims  map[animRef]*figani.Animation
	figani []int
}

func (e *exporter) seed() uint64 { return e.rng.Uint64() >> 12 }

func (e *exporter) load(ref animRef) *figani.Animation {
	if a, ok := e.anims[ref]; ok {
		return a
	}
	root := filepath.Join(e.pack, "animations")
	var a *figani.Animation
	var err error
	if ref.Src == "FIGANI" {
		a, err = figani.LoadSeparatedResource(root, ref.ID)
	} else {
		a, err = figani.LoadSeparatedArchiveResource(root, ref.Src+".DAT", ref.ID)
	}
	if err != nil {
		panic(fmt.Sprintf("%s:%d: %v", ref.Src, ref.ID, err))
	}
	e.dirs[fmt.Sprintf("animations/%s_%03d", ref.Src, ref.ID)] = true
	e.anims[ref] = a
	return a
}

func refJ(r animRef) obj { return obj{"src": r.Src, "id": r.ID} }

func (e *exporter) pickFigani() animRef {
	return animRef{"FIGANI", e.figani[e.rng.IntN(len(e.figani))]}
}

// frameRef 選一個動畫與其中一幀。
func (e *exporter) pickFrame() (animRef, int) {
	ref := e.pickFigani()
	return ref, e.rng.IntN(len(e.load(ref).Frames))
}

func (e *exporter) add(op string, input obj, result obj) {
	e.cases = append(e.cases, obj{"index": len(e.cases), "op": op, "input": input, "result": result})
}

func errJ(err error) obj { return obj{"error": err.Error()} }

func layersJ[T any](layers []T, f func(T) []int) []any {
	out := make([]any, 0, len(layers))
	for _, l := range layers {
		out = append(out, f(l))
	}
	return out
}

func effectLayers(layers []battlepresent.NativeCommandEffectSource) []any {
	return layersJ(layers, func(l battlepresent.NativeCommandEffectSource) []int { return []int{l.Frame, l.X, l.Y} })
}

func srcJ(s battlepresent.NativeCommandSources) obj {
	return obj{"base": digest(s.Base), "actor": s.ActorFrame, "ax": s.ActorOffsetX, "ti": s.TargetIndex,
		"tf": s.TargetFrame, "tx": s.TargetOffsetX, "ty": s.TargetOffsetY, "before": effectLayers(s.Before),
		"after": effectLayers(s.After), "phase": s.Phase, "tbe": s.TargetBeforeEffects, "omit": s.OmitTarget,
		"shader": s.PackedTargetShader}
}

func shas(frames [][]byte) []any {
	out := make([]any, 0, len(frames))
	for _, f := range frames {
		out = append(out, digest(f))
	}
	return out
}

func srcsJ(sources []battlepresent.NativeCommandSources) []any {
	out := make([]any, 0, len(sources))
	for _, s := range sources {
		out = append(out, srcJ(s))
	}
	return out
}

// effectResource 是各指令依 raw side 選的 FDOTHER effect bank（figani Build*Schedule 的同一張表）。
func effectResource(command int, side byte) int {
	table := map[int][2]int{0: {18, 20}, 1: {19, 21}, 2: {26, 27}, 3: {39, 43}, 4: {22, 23}, 5: {24, 25},
		6: {32, 33}, 7: {37, 38}, 8: {28, 30}, 9: {44, 44}}
	pair := table[command]
	if side == 0 {
		return pair[1]
	}
	return pair[0]
}

func (e *exporter) effect(command int, side byte) (animRef, *figani.Animation) {
	ref := animRef{"FDOTHER", effectResource(command, side)}
	return ref, e.load(ref)
}

func (e *exporter) side() byte { return []byte{0, 1, 2}[e.rng.IntN(3)] }

func (e *exporter) genCommand0() {
	for i := 0; i < 24; i++ {
		side := e.side()
		effRef, effect := e.effect(0, side)
		tRef, tFrame := e.pickFrame()
		step, seed := e.rng.IntN(30)-1, e.seed()
		in := obj{"side": side, "effect": refJ(effRef), "target": refJ(tRef), "frame": tFrame, "step": step, "base": seed}
		schedule, err := figani.BuildNativeCommand0PresentationSchedule(side, effect)
		if err != nil {
			e.add("cmd0_target", in, errJ(err))
			continue
		}
		px, err := battlepresent.ComposeNativeCommand0TargetFrame(noise(seed, 64000), e.load(tRef).Frames[tFrame], effect, schedule, step)
		if err != nil {
			e.add("cmd0_target", in, errJ(err))
			continue
		}
		e.add("cmd0_target", in, obj{"px": digest(px)})
	}
	for i := 0; i < 10; i++ {
		side := e.side()
		aRef, tRef := e.pickFigani(), e.pickFigani()
		before, after, bg, plat, lut := e.seed(), e.seed(), e.seed(), e.seed(), e.seed()
		// 真實 FIGANI 動畫多半沒有 raw byte4 marker；大部分案例在複本上把第 marker 幀標成 1（-1 表示不改）。
		actor := *e.load(aRef)
		actor.Frames = append([]figani.Frame(nil), actor.Frames...)
		marker := -1
		if i%4 != 3 {
			marker = e.rng.IntN(len(actor.Frames))
			actor.Frames[marker].RawByte4 = 1
		}
		in := obj{"side": side, "actor": refJ(aRef), "target": refJ(tRef), "before": before, "after": after,
			"bg": bg, "platform": plat, "lut": lut, "marker": marker}
		frames, err := battlepresent.BuildNativeCommand0ActorFrames(battlepresent.NativeCommand0ActorInput{
			BaseBefore: noise(before, 64000), BaseAfter: noise(after, 64000), ActorEffect: &actor,
			FirstTargetIdle: e.load(tRef), RawSide: side, Background: rleFrame(bg, 320, 100),
			Platform: rleFrame(plat, 40, 20), LUT: noise(lut, 256),
		})
		if err != nil {
			e.add("cmd0_actor", in, errJ(err))
			continue
		}
		out := make([]any, 0, len(frames))
		for _, f := range frames {
			out = append(out, obj{"px": digest(f.Pixels), "ef": f.EffectFrame, "er": f.EffectRepeat,
				"tf": f.TargetIdleFrame, "mp": f.PublishMP, "pulse": f.Pulse})
		}
		e.add("cmd0_actor", in, obj{"frames": out})
	}
}

func c1srcJ(s battlepresent.NativeCommand1FrameSources) obj {
	f := func(l battlepresent.NativeCommand1EffectLayer) []int { return []int{l.Frame, l.X, l.Y} }
	return obj{"base": digest(s.Base), "actor": s.ActorFrame, "tf": s.TargetFrame, "tx": s.TargetOffsetX,
		"next": s.NextTarget, "before": layersJ(s.BeforeTarget, f), "after": layersJ(s.AfterTarget, f)}
}

func (e *exporter) genCommand1() {
	for i := 0; i < 6; i++ {
		side := e.side()
		effRef, effect := e.effect(1, side)
		tRef := e.pickFigani()
		seeds := make([]any, 9)
		bases := make([][]byte, 9)
		for k := range bases {
			s := e.seed()
			seeds[k], bases[k] = s, noise(s, 64000)
		}
		in := obj{"side": side, "effect": refJ(effRef), "target": refJ(tRef), "bases": seeds}
		schedule, err := figani.BuildNativeCommand1PresentationSchedule(side, effect)
		if err != nil {
			e.add("cmd1_seq", in, errJ(err))
			continue
		}
		seq, err := battlepresent.BuildNativeCommand1TargetSequence(bases, e.load(tRef), effect, schedule)
		if err != nil {
			e.add("cmd1_seq", in, errJ(err))
			continue
		}
		srcs := make([]any, 0, len(seq.Sources))
		for _, s := range seq.Sources {
			srcs = append(srcs, c1srcJ(s))
		}
		e.add("cmd1_seq", in, obj{"frames": shas(seq.Frames), "src": srcs, "hp": seq.HPStages,
			"samples": seq.SampleMarkers, "next": []int{seq.NextIdleFrame, seq.NextIdleRepeat}})
	}
	for i := 0; i < 6; i++ {
		side := e.side()
		aRef, cRef, nRef, seed := e.pickFigani(), e.pickFigani(), e.pickFigani(), e.seed()
		in := obj{"side": side, "actor": refJ(aRef), "current": refJ(cRef), "next": refJ(nRef), "base": seed}
		seq, err := battlepresent.BuildNativeCommand1TransitionSequence(noise(seed, 64000), e.load(aRef), e.load(cRef), e.load(nRef), side)
		if err != nil {
			e.add("cmd1_trans", in, errJ(err))
			continue
		}
		srcs := make([]any, 0, len(seq.Sources))
		for _, s := range seq.Sources {
			srcs = append(srcs, c1srcJ(s))
		}
		e.add("cmd1_trans", in, obj{"frames": shas(seq.Frames), "src": srcs})
	}
}

func (e *exporter) genCommand2() {
	for i := 0; i < 12; i++ {
		side := e.side()
		// 一成的案例故意用另一側的 schedule，走 raw side 不符的拒收。
		schedSide := side
		if e.rng.IntN(10) == 0 {
			schedSide = 1 - min(side, 1)
		}
		effRef, effect := e.effect(2, side)
		tRef, tFrame := e.pickFrame()
		state, seed := e.rng.IntN(19)-1, e.seed()
		in := obj{"side": side, "sched_side": schedSide, "effect": refJ(effRef), "target": refJ(tRef), "frame": tFrame,
			"state": state, "base": seed}
		_, schedEffect := e.effect(2, schedSide)
		schedule, err := figani.BuildNativeCommand2PresentationSchedule(schedSide, schedEffect)
		if err != nil {
			e.add("cmd2_target", in, errJ(err))
			continue
		}
		px, src, err := battlepresent.ComposeNativeCommand2TargetFrameWithSources(noise(seed, 64000), e.load(tRef).Frames[tFrame], effect, schedule, side, state)
		if err != nil {
			e.add("cmd2_target", in, errJ(err))
			continue
		}
		e.add("cmd2_target", in, obj{"px": digest(px), "src": srcJ(src)})
	}
	for i := 0; i < 12; i++ {
		side := e.side()
		effRef, effect := e.effect(2, side)
		aRef, tRef := e.pickFigani(), e.pickFigani()
		helper, offset, seed := e.rng.IntN(20)-1, e.rng.IntN(281)-140, e.seed()
		transition := i%2 == 1
		in := obj{"side": side, "effect": refJ(effRef), "actor": refJ(aRef), "target": refJ(tRef),
			"helper": helper, "offset": offset, "base": seed, "transition": transition}
		h := figani.NativeCommand2HelperFrame{EffectFrame: helper}
		var px []byte
		var src battlepresent.NativeCommandSources
		var err error
		if transition {
			px, src, err = battlepresent.ComposeNativeCommand2TransitionFrameWithSources(noise(seed, 64000), e.load(aRef), e.load(tRef), effect, h, side, offset)
		} else {
			px, src, err = battlepresent.ComposeNativeCommand2OrbitFrameWithSources(noise(seed, 64000), e.load(aRef), e.load(tRef), effect, h, side)
		}
		if err != nil {
			e.add("cmd2_orbit", in, errJ(err))
			continue
		}
		e.add("cmd2_orbit", in, obj{"px": digest(px), "src": srcJ(src)})
	}
}

func (e *exporter) genCommand24() {
	for i := 0; i < 8; i++ {
		layerSeeds := []uint64{e.seed(), e.seed(), e.seed()}
		source, target := e.seed(), e.seed()
		tRef, tFrame := e.pickFrame()
		right := i%2 == 1
		broken := e.rng.IntN(6) == 0
		in := obj{"layers": layerSeeds, "source": source, "target": target, "idle": refJ(tRef), "frame": tFrame,
			"right": right, "broken": broken}
		layers := [3]fdother.Frame{rleFrame(layerSeeds[0], 320, 100), rleFrame(layerSeeds[1], 320, 100), rleFrame(layerSeeds[2], 320, 100)}
		if broken {
			layers[2] = rleFrame(layerSeeds[2], 320, 99)
		}
		bg := battlepresent.NativeCommand24BackgroundInputs{Layers: layers, Source: noise(source, 64000),
			Target: noise(target, 64000), TargetIdle: e.load(tRef).Frames[tFrame]}
		var presents []battlepresent.NativeCommand24BackgroundPresent
		var err error
		if right {
			presents, err = battlepresent.BuildNativePhysicalRightBackgroundPresents(bg)
		} else {
			presents, err = battlepresent.BuildNativeCommand24BackgroundPresents(bg)
		}
		if err != nil {
			e.add("cmd24_bg", in, errJ(err))
			continue
		}
		out := make([]any, 0, len(presents))
		for _, p := range presents {
			out = append(out, obj{"px": digest(p.Pixels), "in": p.Incoming, "cx": p.ContentX, "bgx": p.BGX,
				"vx": p.ViewportX, "bg": p.BGLayers})
		}
		e.add("cmd24_bg", in, obj{"presents": out})
	}
	for i := 0; i < 12; i++ {
		mode := byte(e.rng.IntN(2))
		side := e.side()
		aRef, aFrame := e.pickFrame()
		tRef, tFrame := e.pickFrame()
		base, plat, dac := e.seed(), e.seed(), e.seed()
		withPlatform := e.rng.IntN(5) != 0
		in := obj{"mode": mode, "side": side, "actor": refJ(aRef), "aframe": aFrame, "target": refJ(tRef),
			"tframe": tFrame, "base": base, "platform": plat, "with_platform": withPlatform, "dac": dac}
		dacBytes := noise(dac, 768)
		for k := range dacBytes {
			dacBytes[k] &= 63
		}
		var platform *fdother.Frame
		if withPlatform {
			p := rleFrame(plat, 40, 20)
			platform = &p
		}
		frames, err := battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{
			Base: noise(base, 64000), ActorIdle: e.load(aRef).Frames[aFrame], FirstTargetIdle: e.load(tRef).Frames[tFrame],
			Platform: platform, RawSide: side, Mode: mode, BaselineDAC: dacBytes,
		})
		if err != nil {
			e.add("prelude", in, errJ(err))
			continue
		}
		out := make([]any, 0, len(frames))
		for _, f := range frames {
			out = append(out, obj{"stage": f.Stage, "px": digest(f.Pixels), "dac": digest(f.DAC)})
		}
		e.add("prelude", in, obj{"frames": out})
	}
}

// seqInput 是 cmd3／5／6／7／8 共用的序列輸入：每個目標 stages+1 張底圖、目標數-1 張轉場底圖。
type seqInput struct {
	in              obj
	front, tail     []byte
	targetBases     [][][]byte
	transitionBases [][]byte
	actor           *figani.Animation
	idles           []*figani.Animation
	effect          *figani.Animation
	side            byte
}

func (e *exporter) seqInput(command int, targets, stages int) seqInput {
	side := e.side()
	if command == 9 && e.rng.IntN(4) != 0 {
		side = 0 // command9 AI 只有 raw side 0 的 selector；偶爾保留非 0 走拒收
	}
	effRef, effect := e.effect(command, side)
	aRef := e.pickFigani()
	front, tail := e.seed(), e.seed()
	s := seqInput{front: noise(front, 64000), tail: noise(tail, 64000), actor: e.load(aRef), effect: effect, side: side}
	idleRefs := make([]any, 0, targets)
	targetSeeds := make([]any, 0, targets)
	for t := 0; t < targets; t++ {
		ref := e.pickFigani()
		idleRefs = append(idleRefs, refJ(ref))
		s.idles = append(s.idles, e.load(ref))
		seeds := make([]any, 0, stages+1)
		bases := make([][]byte, 0, stages+1)
		for k := 0; k <= stages; k++ {
			v := e.seed()
			seeds, bases = append(seeds, v), append(bases, noise(v, 64000))
		}
		targetSeeds = append(targetSeeds, seeds)
		s.targetBases = append(s.targetBases, bases)
	}
	transSeeds := make([]any, 0, targets)
	for t := 0; t+1 < targets; t++ {
		v := e.seed()
		transSeeds, s.transitionBases = append(transSeeds, v), append(s.transitionBases, noise(v, 64000))
	}
	s.in = obj{"side": side, "effect": refJ(effRef), "actor": refJ(aRef), "idles": idleRefs, "front": front,
		"tail": tail, "target_bases": targetSeeds, "transition_bases": transSeeds}
	return s
}

func renderedJ(px []byte, hp int, p1, p2 bool, src battlepresent.NativeCommandSources) obj {
	return obj{"px": digest(px), "hp": hp, "p1": p1, "p2": p2, "src": srcJ(src)}
}

func listJ[T any](items []T, f func(T) obj) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		out = append(out, f(it))
	}
	return out
}

func nestedJ[T any](items [][]T, f func(T) obj) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		out = append(out, listJ(it, f))
	}
	return out
}

func (e *exporter) genSequences() {
	for i := 0; i < 6; i++ {
		s := e.seqInput(3, 1+e.rng.IntN(3), figani.NativeCommand3DamageStages)
		schedule, err := figani.BuildNativeCommand3PresentationSchedule(s.side, s.effect)
		if err != nil {
			e.add("cmd3_seq", s.in, errJ(err))
			continue
		}
		seq, err := battlepresent.BuildNativeCommand3EffectSequence(battlepresent.NativeCommand3EffectInput{
			FrontBase: s.front, TailBase: s.tail, TargetBases: s.targetBases, TransitionBases: s.transitionBases,
			ActorEffect: s.actor, TargetIdle: s.idles, Effect: s.effect, Schedule: schedule, RawSide: s.side})
		if err != nil {
			e.add("cmd3_seq", s.in, errJ(err))
			continue
		}
		f := func(r battlepresent.NativeCommand3RenderedFrame) obj {
			return renderedJ(r.Pixels, r.HPStage, r.PlaySub1, r.PlaySub2, r.Sources)
		}
		e.add("cmd3_seq", s.in, obj{"front": listJ(seq.Front, f), "targets": nestedJ(seq.Targets, f),
			"transitions": nestedJ(seq.Transitions, f), "tail": listJ(seq.Tail, f),
			"next": []int{seq.NextIdleFrame, seq.NextIdleRepeat}})
	}
	for i := 0; i < 8; i++ {
		command := 5
		if i%3 == 2 {
			command = 4
		}
		s := e.seqInput(command, 1+e.rng.IntN(3), figani.NativeCommand5DamageStages)
		visual := uint16(e.rng.IntN(65536))
		resolve := i%2 == 0
		s.in["command"], s.in["visual_rng"], s.in["resolve"] = command, visual, resolve
		var schedule figani.NativeCommand5PresentationSchedule
		var err error
		if command == 4 {
			schedule, err = figani.BuildNativeCommand4PresentationSchedule(s.side, s.effect)
		} else {
			schedule, err = figani.BuildNativeCommand5PresentationSchedule(s.side, s.effect)
		}
		if err != nil {
			e.add("cmd5_seq", s.in, errJ(err))
			continue
		}
		in := battlepresent.NativeCommand5EffectInput{
			FrontBase: s.front, TailBase: s.tail, TargetBases: s.targetBases, TransitionBases: s.transitionBases,
			ActorEffect: s.actor, TargetIdle: s.idles, Effect: s.effect, Schedule: schedule, RawSide: s.side, VisualRNG: visual}
		if resolve {
			// 兩端共用的確定性擲骰：next = rng*75+74+index（uint16 繞回），hit = bit3 為 0。
			in.ResolveTarget = func(index int, rng uint16) (uint16, bool, error) {
				return rng*75 + 74 + uint16(index), (rng>>3)&1 == 0, nil
			}
		}
		seq, err := battlepresent.BuildNativeCommand5EffectSequence(in)
		if err != nil {
			e.add("cmd5_seq", s.in, errJ(err))
			continue
		}
		f := func(r battlepresent.NativeCommand5RenderedFrame) obj {
			src := r.Sources
			layers := layersJ(src.Layers, func(l figani.NativeCommand5Layer) []int { return []int{l.Channel, l.Frame, l.X} })
			return obj{"px": digest(r.Pixels), "hp": r.HPStage, "p1": r.PlayPrimary, "p2": r.PlaySecondary,
				"src": obj{"base": digest(src.Base), "actor": src.ActorFrame, "ti": src.TargetIndex, "tf": src.TargetFrame,
					"tx": src.TargetOffsetX, "layers": layers, "phase": src.Phase}}
		}
		e.add("cmd5_seq", s.in, obj{"front": listJ(seq.Front, f), "targets": nestedJ(seq.Targets, f),
			"transitions": nestedJ(seq.Transitions, f), "tail": listJ(seq.Tail, f), "rng": seq.VisualRNG,
			"next": []int{seq.NextIdleFrame, seq.NextIdleRepeat}})
	}
	for i := 0; i < 6; i++ {
		targets := 1 + e.rng.IntN(3)
		s := e.seqInput(6, targets, figani.NativeCommand6DamageStages)
		hits := make([]bool, targets)
		numeric := make([]uint16, targets)
		for t := range hits {
			hits[t], numeric[t] = e.rng.IntN(2) == 0, uint16(e.rng.IntN(65536))
		}
		s.in["hits"], s.in["numeric_rng"] = hits, numeric
		schedule, err := figani.BuildNativeCommand6PresentationSchedule(s.side, s.effect)
		if err != nil {
			e.add("cmd6_seq", s.in, errJ(err))
			continue
		}
		seq, err := battlepresent.BuildNativeCommand6EffectSequence(battlepresent.NativeCommand6EffectInput{
			TargetHits: hits, TargetNumericRNG: numeric, FrontBase: s.front, TailBase: s.tail, TargetBases: s.targetBases,
			TransitionBases: s.transitionBases, ActorEffect: s.actor, TargetIdle: s.idles, Effect: s.effect,
			Schedule: schedule, RawSide: s.side})
		if err != nil {
			e.add("cmd6_seq", s.in, errJ(err))
			continue
		}
		targetsJ := make([]any, 0, len(seq.Targets))
		for _, t := range seq.Targets {
			targetsJ = append(targetsJ, obj{"frames": shas(t.Frames), "hp": t.HPStages, "src": srcsJ(t.Sources)})
		}
		trans := make([]any, 0, len(seq.Transitions))
		for _, t := range seq.Transitions {
			trans = append(trans, shas(t))
		}
		transSrc := make([]any, 0, len(seq.TransitionSources))
		for _, t := range seq.TransitionSources {
			transSrc = append(transSrc, srcsJ(t))
		}
		e.add("cmd6_seq", s.in, obj{"rng": seq.RNGAfter, "front": shas(seq.Front), "front_src": srcsJ(seq.FrontSources),
			"targets": targetsJ, "transitions": trans, "transition_src": transSrc, "tail": shas(seq.Tail),
			"tail_src": srcsJ(seq.TailSources)})
	}
	for i := 0; i < 6; i++ {
		s := e.seqInput(7, 1+e.rng.IntN(3), figani.NativeCommand7DamageStages)
		schedule, err := figani.BuildNativeCommand7PresentationSchedule(s.side, s.effect)
		if err != nil {
			e.add("cmd7_seq", s.in, errJ(err))
			continue
		}
		seq, err := battlepresent.BuildNativeCommand7EffectSequence(battlepresent.NativeCommand7EffectInput{
			FrontBase: s.front, TailBase: s.tail, TargetBases: s.targetBases, TransitionBases: s.transitionBases,
			ActorEffect: s.actor, TargetIdle: s.idles, Effect: s.effect, Schedule: schedule, RawSide: s.side})
		if err != nil {
			e.add("cmd7_seq", s.in, errJ(err))
			continue
		}
		f := func(r battlepresent.NativeCommand7RenderedFrame) obj {
			return renderedJ(r.Pixels, r.HPStage, r.PlayHandleA, r.PlayHandleB, r.Sources)
		}
		e.add("cmd7_seq", s.in, obj{"front": listJ(seq.Front, f), "targets": nestedJ(seq.Targets, f),
			"transitions": nestedJ(seq.Transitions, f), "tail": listJ(seq.Tail, f),
			"next": []int{seq.NextIdleFrame, seq.NextIdleRepeat}})
	}
	for i := 0; i < 4; i++ {
		s := e.seqInput(8, 1, figani.NativeCommand8DamageStages)
		schedule, err := figani.BuildNativeCommand8PresentationSchedule(s.side, s.effect)
		if err != nil {
			e.add("cmd8_seq", s.in, errJ(err))
			continue
		}
		seq, err := battlepresent.BuildNativeCommand8EffectSequence(battlepresent.NativeCommand8EffectInput{
			FrontBase: s.front, TailBase: s.tail, TargetBases: s.targetBases, TransitionBases: s.transitionBases,
			ActorEffect: s.actor, TargetIdle: s.idles, Effect: s.effect, Schedule: schedule, RawSide: s.side})
		if err != nil {
			e.add("cmd8_seq", s.in, errJ(err))
			continue
		}
		f := func(r battlepresent.NativeCommand8RenderedFrame) obj {
			return renderedJ(r.Pixels, r.HPStage, r.PlaySub1, r.PlaySub2, r.Sources)
		}
		e.add("cmd8_seq", s.in, obj{"front": listJ(seq.Front, f), "targets": nestedJ(seq.Targets, f),
			"transitions": nestedJ(seq.Transitions, f), "tail": listJ(seq.Tail, f),
			"next": []int{seq.NextIdleFrame, seq.NextIdleRepeat}})
	}
	for i := 0; i < 4; i++ {
		s := e.seqInput(9, 1, figani.NativeCommand9AIDamageStages)
		tRef := e.pickFigani()
		s.in["target"] = refJ(tRef)
		schedule, err := figani.BuildNativeCommand9AISchedule(s.side, s.effect)
		if err != nil {
			e.add("cmd9_seq", s.in, errJ(err))
			continue
		}
		seq, err := battlepresent.BuildNativeCommand9AISequence(battlepresent.NativeCommand9AIInput{
			BeforeBase: s.front, AfterBase: s.tail, TargetBases: s.targetBases[0], Actor: s.actor,
			Target: e.load(tRef), Effect: s.effect, Schedule: schedule})
		if err != nil {
			e.add("cmd9_seq", s.in, errJ(err))
			continue
		}
		f := func(r battlepresent.NativeCommand9RenderedFrame) obj {
			return renderedJ(r.Pixels, r.HPStage, r.PlaySub1, r.PlaySub2, r.Sources)
		}
		e.add("cmd9_seq", s.in, obj{"slide_in": listJ(seq.SlideIn, f), "front": listJ(seq.Front, f),
			"target": listJ(seq.Target, f), "tail": listJ(seq.Tail, f), "slide_out": listJ(seq.SlideOut, f),
			"next": []int{seq.NextIdleFrame, seq.NextIdleRepeat}})
	}
}

func (e *exporter) genCompound() {
	for command := 32; command <= 35; command++ {
		for k := 0; k < 2; k++ {
			ref := animRef{"FDOTHER", command + 33}
			effect := e.load(ref)
			aRef, aFrame := e.pickFrame()
			seed := e.seed()
			in := obj{"command": command, "effect": refJ(ref), "actor": refJ(aRef), "frame": aFrame, "base": seed}
			schedule, err := figani.BuildNativeCompoundPresentationSchedule(command, effect)
			if err != nil {
				e.add("compound_common", in, errJ(err))
				continue
			}
			frames, err := battlepresent.BuildNativeCompoundCommonFrames(noise(seed, 64000), e.load(aRef).Frames[aFrame], effect, schedule)
			if err != nil {
				e.add("compound_common", in, errJ(err))
				continue
			}
			e.add("compound_common", in, obj{"actor": shas(frames.ActorSlide), "effect": shas(frames.EffectSlide),
				"main": shas(frames.Main), "tail": shas(frames.Tail)})
		}
	}
	for i := 0; i < 8; i++ {
		side := e.side()
		aRef, tRef, seed := e.pickFigani(), e.pickFigani(), e.seed()
		in := obj{"side": side, "actor": refJ(aRef), "target": refJ(tRef), "base": seed}
		presents, err := battlepresent.BuildNativeCompoundActorPresents(noise(seed, 64000), e.load(aRef), e.load(tRef), side)
		if err != nil {
			e.add("compound_actor", in, errJ(err))
			continue
		}
		out := make([]any, 0, len(presents))
		for _, p := range presents {
			out = append(out, obj{"px": digest(p.Pixels), "ai": p.ActorIndex, "ti": p.TargetIndex})
		}
		e.add("compound_actor", in, obj{"presents": out})
	}
}

func tailLayersJ(groups [][]indexedmap.NativeCommandTailLayer) []any {
	out := make([]any, 0, len(groups))
	for _, g := range groups {
		out = append(out, layersJ(g, func(l indexedmap.NativeCommandTailLayer) []int { return []int{l.Resource, l.Entry, l.X, l.Y} }))
	}
	return out
}

func (e *exporter) genTails() {
	for i := 0; i < 10; i++ {
		ai := i%2 == 1
		command := []int{1, 6, 7}[e.rng.IntN(3)]
		effSeed, digitSeed := e.seed(), e.seed()
		effCount, digitCount := 0x4c+e.rng.IntN(4), 0x77+e.rng.IntN(8)
		if e.rng.IntN(8) == 0 {
			digitCount = 0x70 // 缺 miss 描述子 0x74..0x76：走拒收
		}
		if e.rng.IntN(6) == 0 {
			effCount = 0x4b // 少一個 toggle 描述子：走拒收
		}
		work, vga, postWork, postVGA := e.seed(), e.seed(), e.seed(), e.seed()
		cameraX, cameraY := e.rng.IntN(4), e.rng.IntN(4)
		n := 1 + e.rng.IntN(4)
		targetsJ, resultsJ := make([]any, 0, n), make([]any, 0, n)
		targets := make([]indexedmap.NativeCommandHealTailTarget, 0, n)
		results := make([]battlepresent.NativeCommand32TailResult, 0, n)
		for t := 0; t < n; t++ {
			target := indexedmap.NativeCommandHealTailTarget{RecordIndex: 2*t + e.rng.IntN(2), X: cameraX + e.rng.IntN(14) - 1,
				Y: cameraY + e.rng.IntN(9) - 1}
			result := battlepresent.NativeCommand32TailResult{TargetRecord: target.RecordIndex, Hit: e.rng.IntN(3) != 0,
				Damage: e.rng.IntN(1200)}
			targets, results = append(targets, target), append(results, result)
			targetsJ = append(targetsJ, []int{target.RecordIndex, target.X, target.Y, target.SelectorSlot})
			resultsJ = append(resultsJ, obj{"record": result.TargetRecord, "hit": result.Hit, "damage": result.Damage})
		}
		in := obj{"ai": ai, "command": command, "effect": effSeed, "digits": digitSeed, "effect_count": effCount,
			"digit_count": digitCount, "work": work, "vga": vga, "post_work": postWork, "post_vga": postVGA,
			"camera": []int{cameraX, cameraY}, "targets": targetsJ, "results": resultsJ}
		effect, digits := lmiList(effSeed, effCount), lmiList(digitSeed, digitCount)
		w, v := noise(work, indexedmap.NativeUnitPresentWorkSize), noise(vga, indexedmap.NativeMapVGASize)
		pw, pv := noise(postWork, indexedmap.NativeUnitPresentWorkSize), noise(postVGA, indexedmap.NativeMapVGASize)
		var frames battlepresent.NativeCommand32TailFrames
		var err error
		if ai {
			var schedule fdother.NativeAIItemType21TailSchedule
			schedule, err = fdother.BuildNativeAIItemType21TailSchedule(command, effect, digits)
			if err == nil {
				frames, err = battlepresent.BuildNativeAIItemType21TailFrames(w, v, pw, pv, effect, digits, targets, results, cameraX, cameraY, schedule)
			}
		} else {
			var schedule fdother.NativeCommand32TailSchedule
			schedule, err = fdother.BuildNativeCommand32TailSchedule(effect, digits)
			if err == nil {
				frames, err = battlepresent.BuildNativeCommand32TailFrames(w, v, pw, pv, effect, digits, targets, results, cameraX, cameraY, schedule)
			}
		}
		if err != nil {
			e.add("tail", in, errJ(err))
			continue
		}
		queue := layersJ(frames.Queue, func(d battle.NativePresentationDigit) []int { return []int{d.PositionCode, d.Target, d.Digit} })
		e.add("tail", in, obj{"effect": shas(frames.Effect), "toggle": shas(frames.Toggle), "result": shas(frames.Result),
			"queue": queue, "effect_src": tailLayersJ(frames.EffectSources), "toggle_src": tailLayersJ(frames.ToggleSources),
			"result_src": tailLayersJ(frames.ResultSources)})
	}
}

func (e *exporter) genPhysical() {
	for i := 0; i < 12; i++ {
		side := e.side()
		aRef, iRef := e.pickFigani(), e.pickFigani()
		hp := e.rng.IntN(300)
		n := 1 + e.rng.IntN(3)
		strikes := make([]battle.NativePhysicalStrike, 0, n)
		strikesJ := make([]any, 0, n)
		for k := 0; k < n; k++ {
			roll := battle.NativePhysicalRollResult{Missed: e.rng.IntN(4) == 0, Crit: e.rng.IntN(3) == 0,
				Status: e.rng.IntN(3) == 0, Damage: e.rng.IntN(150) - 5}
			s := battle.NativePhysicalStrike{Roll: roll, DefenderHP: e.rng.IntN(200) - 2}
			strikes = append(strikes, s)
			strikesJ = append(strikesJ, obj{"missed": roll.Missed, "crit": roll.Crit, "status": roll.Status,
				"damage": roll.Damage, "defender_hp": s.DefenderHP})
		}
		in := obj{"side": side, "attack": refJ(aRef), "idle": refJ(iRef), "hp": hp, "strikes": strikesJ}
		plan, err := battlepresent.BuildNativePhysicalBodyPlan(e.load(aRef), e.load(iRef), side, hp, strikes)
		if err != nil {
			e.add("phys_plan", in, errJ(err))
			continue
		}
		out := make([]any, 0, len(plan))
		for _, f := range plan {
			presents := make([]any, 0, len(f.Presents))
			for _, p := range f.Presents {
				presents = append(presents, []any{p.IdleFrame, p.DX, p.DY, p.OpaqueFill, p.StatusPulse, p.CritPulse})
			}
			out = append(out, obj{"strike": f.Strike, "frame": f.Frame, "hp": f.HP, "panel": f.UpdatePanel,
				"cue": f.Cue, "presents": presents})
		}
		e.add("phys_plan", in, obj{"plan": out})
	}
	for i := 0; i < 12; i++ {
		side := e.side()
		aRef, aFrame := e.pickFrame()
		iRef, iFrame := e.pickFrame()
		fill := []int{-1, 33, 5}[e.rng.IntN(3)]
		dx, dy, seed := e.rng.IntN(37)-18, e.rng.IntN(21)-10, e.seed()
		in := obj{"side": side, "attack": refJ(aRef), "aframe": aFrame, "idle": refJ(iRef), "iframe": iFrame,
			"fill": fill, "dx": dx, "dy": dy, "base": seed}
		px, err := battlepresent.ComposeNativePhysicalBodyPresent(noise(seed, 64000), e.load(aRef).Frames[aFrame],
			e.load(iRef).Frames[iFrame], side, battlepresent.NativePhysicalBodyPresent{DX: dx, DY: dy, OpaqueFill: fill})
		if err != nil {
			e.add("phys_present", in, errJ(err))
			continue
		}
		e.add("phys_present", in, obj{"px": digest(px)})
	}
}

// recordDirs 把用到的動畫目錄裡每個檔案（排序後）連同大小與 SHA-256 記下。
func (e *exporter) recordDirs() {
	dirs := make([]string, 0, len(e.dirs))
	for d := range e.dirs {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		var names []string
		err := filepath.WalkDir(filepath.Join(e.pack, d), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() {
				rel, _ := filepath.Rel(e.pack, path)
				names = append(names, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
		sort.Strings(names)
		for _, name := range names {
			raw, err := os.ReadFile(filepath.Join(e.pack, name))
			if err != nil {
				panic(err)
			}
			e.files = append(e.files, obj{"path": "pack/" + name, "size": len(raw), "sha256": digest(raw)})
		}
	}
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	pack := flag.String("pack", "", "分離素材包根目錄（fd2-original-b97caf22）")
	flag.Parse()
	if *out == "" || *pack == "" {
		fmt.Fprintln(os.Stderr, "需要 -out、-pack")
		os.Exit(2)
	}
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x42505245)), pack: *pack, dirs: map[string]bool{},
		anims: map[animRef]*figani.Animation{}}
	entries, err := os.ReadDir(filepath.Join(*pack, "animations"))
	if err != nil {
		panic(err)
	}
	pattern := regexp.MustCompile(`^FIGANI_(\d{3})$`)
	var all []int
	for _, entry := range entries {
		if m := pattern.FindStringSubmatch(entry.Name()); m != nil {
			id, _ := strconv.Atoi(m[1])
			all = append(all, id)
		}
	}
	sort.Ints(all)
	// 固定種子挑 20 個能載入、至少一幀的 FIGANI 動畫當角色與目標。
	for len(e.figani) < 20 {
		id := all[e.rng.IntN(len(all))]
		if a, err := figani.LoadSeparatedResource(filepath.Join(*pack, "animations"), id); err == nil && len(a.Frames) > 0 {
			dup := false
			for _, v := range e.figani {
				dup = dup || v == id
			}
			if !dup {
				e.figani = append(e.figani, id)
			}
		}
	}
	e.genCommand0()
	e.genCommand1()
	e.genCommand2()
	e.genCommand24()
	e.genSequences()
	e.genCompound()
	e.genTails()
	e.genPhysical()
	e.recordDirs()
	doc := obj{"schema": "fd2-golden-present-bp", "version": 1, "seed": *seed, "figani": e.figani,
		"files": e.files, "cases": e.cases}
	raw, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, raw, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("cases=%d files=%d\n", len(e.cases), len(e.files))
}
