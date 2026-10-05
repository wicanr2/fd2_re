package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// #34 enumerates only the opening presents already implemented by the normal
// overlay owner. Contract: docs/data/ida/fd2_parity_ring_open_20261005.json.
type parityActionOverlayCandidate struct {
	Phase        string `json:"phase"`
	Frame        int    `json:"frame"`
	OpenVariant  bool   `json:"open_variant"`
	BlinkPhase   int    `json:"blink_phase"`
	FirstVariant int    `json:"first_variant"`
	VariantCount int    `json:"variant_count"`
}

func TestNativeActionOverlayMapAdmissionRequiresThisDrawAndFaithfulSource(t *testing.T) {
	for _, name := range []string{"admitted", "not_presented", "no_ring", "no_actor", "missing_cells", "modern"} {
		t.Run(name, func(t *testing.T) {
			g := &Game{ring: true, sel: &battle.Unit{}, nativeActionCellsRaw: make([]fdother.RawCell, nativeActionOverlayCellCount)}
			presented := true
			switch name {
			case "not_presented":
				presented = false
				g.nativeMapVGA = make([]byte, 320*200)
			case "no_ring":
				g.ring = false
			case "no_actor":
				g.sel = nil
				g.nativeSystemCursorOverlay = true
			case "missing_cells":
				g.nativeActionCellsRaw = g.nativeActionCellsRaw[:77]
			case "modern":
				g.modernStoryPortraits = &modernStoryPortraitSet{battleActionIcons: make([]image.Image, 4)}
			}
			if got := g.nativeActionOverlayInMapFrame(presented); got != (name == "admitted") {
				t.Fatalf("presented gate=%v", got)
			}
		})
	}
}

func TestNativeActionOverlayCursorRedrawUsesFirstActiveRawRecord(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		inactive, missing, closing bool
	}{
		{"first_active", false, false, false}, {"skip_inactive", true, false, false},
		{"missing_raw", false, true, false}, {"closing", false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assets, _, state := completeNativeMapFrameFixture(t)
			view := battle.NativeMapViewState{CursorX: 4, CursorY: 4, VisibleCursorX: 4, VisibleCursorY: 4}
			if err := state.MaterializeNativeMapViewState(view); err != nil {
				t.Fatal(err)
			}
			first := state.Units[0]
			first.NativeMapPresentation.X, first.NativeMapPresentation.Y = 4, 4
			first.X, first.Y = 9, 9 // Lookup must ignore normalized coordinates and g.sel.
			second := *first
			second.NativeMapPresentation.Pose = 1
			state.Units = append(state.Units, &second)
			for i := range assets.Units.Sprites {
				assets.Units.Sprites[i] = nativeFrameTestSprite(byte(3 + i))
			}
			if tc.inactive {
				first.NativeRecordByte5 |= 1
			}
			if tc.missing {
				first.HasNativeMapPresentation = false
			}
			cells := make([]fdother.RawCell, nativeActionOverlayCellCount)
			for i := range cells {
				cells[i] = fdother.RawCell{Width: 80, Height: 80, Pixels: bytes.Repeat([]byte{240}, 80*80)}
			}
			g := &Game{st: state, sel: &second, ring: true, nativeMapAssets: assets,
				nativeActionCellsRaw: cells, actionOverlayPhase: actionOverlayOpen, actionOverlayFrame: 3}
			if tc.closing {
				g.actionOverlayPhase = actionOverlayClosing
			}
			buf := make([]byte, 456*400)
			err := g.nativeMapActionOverlay(state)(buf, 456, 0)
			if tc.missing {
				if err == nil {
					t.Fatal("missing raw provenance accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			pos, _ := fdicon.NativePlacementOffset(4, 4, 0, 0, 0, 0, 0, false)
			if tc.closing {
				if buf[pos+8*456+8] != 240 {
					t.Fatal("closing acquired cursor redraw")
				}
				return
			}
			chosen := first
			if tc.inactive {
				chosen = &second
			}
			entry, ok := chosen.NativeUnitLayerEntry()
			if !ok {
				t.Fatal("fixture lacks raw entry")
			}
			sprite, err := assets.Units.SpriteForNativeSlot(state.NativeMapSelectorCache, entry.Slot, entry.Pose, state.NativeMapCycleState.Idle)
			if err != nil {
				t.Fatal(err)
			}
			if buf[pos+8*456+8] != sprite.Pixels[8*24+8] {
				t.Fatal("cursor redraw did not use first active raw record")
			}
		})
	}
}

// #180: this prefix reaches the ring through the normal chapter replay. Only
// finite presentation phases and the validated raw DAC-cycle input are varied.
func (r *parityReplay) verifyRingGPU(action parityAction) {
	t, g := r.t, r.g
	sourceRun := r.run
	if supplied := os.Getenv("FD2_PARITY_RING_GPU_ORIGINAL"); supplied != "" {
		sourceRun = supplied
	}
	source := filepath.Join(sourceRun, fmt.Sprintf("checkpoint-%04d.png", action.Seq))
	checkpoint, err := os.ReadFile(filepath.Join(sourceRun, fmt.Sprintf("checkpoint-%04d.json", action.Seq)))
	if err != nil {
		t.Fatal(err)
	}
	var owner struct {
		SHA   string   `json:"exe_sha256"`
		Chain []string `json:"input_chain"`
		View  struct {
			CursorX int `json:"cursor_x"`
			CursorY int `json:"cursor_y"`
		} `json:"view"`
	}
	if json.Unmarshal(checkpoint, &owner) != nil || owner.SHA != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" ||
		!slices.Contains(owner.Chain, "0x17815") || owner.View.CursorX != g.st.NativeMapViewState.CursorX || owner.View.CursorY != g.st.NativeMapViewState.CursorY {
		t.Fatal("GPU original cursor/ring provenance differs")
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	original, ok := decoded.(*image.Paletted)
	if !ok || original.Bounds() != image.Rect(0, 0, 320, 200) {
		t.Fatal("incomplete original indexed frame")
	}
	phase, ok := parityPaletteCyclePhase(original.Palette)
	if !ok {
		t.Fatal("GPU prefix requires a complete validated raw DAC-cycle phase")
	}
	frozen := time.Now()
	g.nativeMapFrozenNow = func() time.Time { return frozen }
	defer func() { g.nativeMapFrozenNow = nil }()
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	saved := parityActionOverlayCandidate{Phase: g.actionOverlayPhase, Frame: g.actionOverlayFrame,
		OpenVariant: g.nativeActionOverlayOpenFrameVariant, BlinkPhase: g.actionOverlayBlink.Phase}
	defer r.applyActionOverlayCandidate(saved)
	probe := &ringGPUProbe{r: r, original: original, phase: phase}
	for _, overlay := range r.actionOverlayCandidates(action.Kind) {
		for flip := 0; flip < 2; flip++ {
			for idle := 0; idle < 4; idle++ {
				probe.jobs = append(probe.jobs, ringGPUJob{overlay: overlay, flip: flip, idle: idle})
			}
		}
	}
	units, _ := json.Marshal(g.st.Units)
	rng := g.nativeRNGState
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	diagnostic, _ := json.MarshalIndent(probe.rows, "", "  ")
	if err := os.WriteFile(filepath.Join(r.out, "ring-gpu-candidates.json"), append(diagnostic, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(g.st.Units)
	if probe.err != "" || probe.index != len(probe.jobs) || probe.zero == 0 ||
		!bytes.Equal(units, after) || rng != g.nativeRNGState {
		t.Fatal("normal ring GPU coverage", probe.err, probe.index, probe.zero)
	}
	r.applyActionOverlayCandidate(saved)
	r.cancelUnit(action, g.sel)
	if !pump(t, g, 600, func() bool { return !g.ring && !g.actionOverlayBlocksInput() }) {
		t.Fatal("normal ring ESC did not restore input")
	}
	encoded, _ := json.MarshalIndent(map[string]any{"oracle_seq": action.Seq, "kind": action.Kind,
		"source": source, "source_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)),
		"source_checkpoint_sha256": fmt.Sprintf("%x", sha256.Sum256(checkpoint)),
		"candidates":               probe.rows, "full_indexed_and_gpu_zero": probe.zero, "semantic_cancel_return": true,
		"method": "normal LOAD/town/selection/movement; actual Game.Draw; finite presentation phases and validated raw DAC phase; no pixel injection; prefix only"}, "", "  ")
	if err := os.WriteFile(filepath.Join(r.out, "ring-gpu.json"), append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

type ringGPUJob struct {
	overlay    parityActionOverlayCandidate
	flip, idle int
}
type ringGPUProbe struct {
	r                         *parityReplay
	original                  *image.Paletted
	jobs                      []ringGPUJob
	phase, index, zero, ticks int
	await                     bool
	err                       string
	rows                      []map[string]any
}

func (p *ringGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *ringGPUProbe) Update() error {
	p.ticks++
	if p.err != "" || p.index == len(p.jobs) {
		return ebiten.Termination
	}
	if p.ticks > 1000 {
		p.err = "GPU draw deadline"
		return ebiten.Termination
	}
	if p.await {
		return nil
	}
	g := p.r.g
	j := p.jobs[p.index]
	p.r.applyActionOverlayCandidate(j.overlay)
	g.st.NativeMapCycleState.Idle = j.idle
	g.st.NativeTerrainFlipState.Value = j.flip
	tick, ok := g.nativeMapClock.Current()
	if !ok {
		p.err = "BIOS input unavailable"
		return ebiten.Termination
	}
	g.nativeFDOTHERPalettePhase, g.nativeFDOTHERPaletteTick = p.phase, tick
	if err := fdother.ApplyNativeDACPaletteCycleE0EF(g.nativeMapDAC, p.phase); err != nil {
		return err
	}
	p.await = true
	return nil
}
func (p *ringGPUProbe) Draw(screen *ebiten.Image) {
	if !p.await || p.err != "" {
		return
	}
	g := p.r.g
	j := p.jobs[p.index]
	screen.Clear()
	g.Draw(screen)
	if g.loadErr != "" {
		p.err = g.loadErr
		return
	}
	rgba := make([]byte, 640*400*4)
	screen.ReadPixels(rgba)
	indexedDiff, gpuDiff := 0, 0
	if len(g.nativeMapVGA) != len(p.original.Pix) {
		p.err = "formal map output absent"
		return
	}
	for i, v := range p.original.Pix {
		if g.nativeMapVGA[i] != v {
			indexedDiff++
		}
	}
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			r, gg, b, a := p.original.At(x/2, y/2).RGBA()
			offset := (y*640 + x) * 4
			if rgba[offset] != byte(r>>8) || rgba[offset+1] != byte(gg>>8) || rgba[offset+2] != byte(b>>8) || rgba[offset+3] != byte(a>>8) {
				gpuDiff++
			}
		}
	}
	name := fmt.Sprintf("ring-gpu-%02d.png", p.index)
	if indexedDiff == 0 && gpuDiff == 0 {
		p.zero++
	}
	if indexedDiff == 0 {
		f, err := os.Create(filepath.Join(p.r.out, name))
		if err != nil {
			p.err = err.Error()
			return
		}
		err = png.Encode(f, &image.RGBA{Pix: rgba, Stride: 640 * 4, Rect: image.Rect(0, 0, 640, 400)})
		f.Close()
		if err != nil {
			p.err = err.Error()
			return
		}
	}
	p.rows = append(p.rows, map[string]any{"overlay": j.overlay, "terrain_flip": j.flip, "idle": j.idle,
		"palette_cycle_phase": p.phase, "indexed_diff_pixels": indexedDiff, "gpu_rgba_diff_pixels": gpuDiff,
		"gpu_rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(rgba)), "file": name})
	p.await = false
	p.index++
}

func (r *parityReplay) actionOverlayCandidates(kind string) []parityActionOverlayCandidate {
	g := r.g
	current := parityActionOverlayCandidate{
		Phase: g.actionOverlayPhase, Frame: g.actionOverlayFrame,
		OpenVariant: g.nativeActionOverlayOpenFrameVariant,
		BlinkPhase:  g.actionOverlayBlink.Phase,
	}
	if !g.ring || g.actionOverlayBlocksInput() {
		return []parityActionOverlayCandidate{current}
	}
	current.OpenVariant = false
	lastOpen := current
	lastOpen.OpenVariant = true
	states := []parityActionOverlayCandidate{current, lastOpen}
	if kind == "move" || kind == "stay" {
		otherBlink := current
		otherBlink.BlinkPhase = 1 - current.BlinkPhase
		states = append(states, otherBlink)
		// The legacy lastOpen candidate already covers frame 3. These three
		// presents complete the finite 0..3 set without duplicating it.
		for frame := 0; frame < 3; frame++ {
			states = append(states, parityActionOverlayCandidate{Phase: actionOverlayOpening, Frame: frame})
		}
	}
	return states
}

func (r *parityReplay) applyActionOverlayCandidate(state parityActionOverlayCandidate) {
	r.g.actionOverlayPhase = state.Phase
	r.g.actionOverlayFrame = state.Frame
	r.g.nativeActionOverlayOpenFrameVariant = state.OpenVariant
	r.g.actionOverlayBlink.Phase = state.BlinkPhase
}

func TestParityRingOpenCandidatesRespectWaitingOwner(t *testing.T) {
	g := &Game{ring: true, actionOverlayPhase: actionOverlayOpen, actionOverlayFrame: 3}
	r := parityReplay{g: g}
	for _, kind := range []string{"move", "stay"} {
		states := r.actionOverlayCandidates(kind)
		opening := map[int]bool{}
		for _, state := range states {
			if state.OpenVariant {
				opening[3] = true
			} else if state.Phase == actionOverlayOpening {
				opening[state.Frame] = true
			}
		}
		if len(states) != 6 || len(opening) != 4 {
			t.Fatalf("%s candidates=%+v, missing one of the four native opening presents", kind, states)
		}
	}
	if got := r.actionOverlayCandidates("attack_result"); len(got) != 2 {
		t.Fatal("unrelated checkpoint acquired extra opening candidates")
	}
	for _, phase := range []string{actionOverlayOpening, actionOverlayClosing} {
		g.actionOverlayPhase, g.actionOverlayFrame = phase, 1
		if got := r.actionOverlayCandidates("move"); len(got) != 1 || got[0].Phase != phase || got[0].Frame != 1 {
			t.Fatal("active lifecycle was replaced by candidate waiting states")
		}
	}
	g.ring = false
	if got := r.actionOverlayCandidates("stay"); len(got) != 1 {
		t.Fatal("non-ring owner acquired opening candidates")
	}
}

func TestParityRingOpenFramesRestoreOwnerAndRepeat(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	if err := state.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CursorX: 4, CursorY: 4, VisibleCursorX: 4, VisibleCursorY: 4,
	}); err != nil {
		t.Fatal(err)
	}
	u := state.Units[0]
	u.X, u.Y, u.Camp, u.NativeRecordByte6 = 4, 4, battle.Own, 2
	cells := make([]fdother.RawCell, nativeActionOverlayCellCount)
	for i := range cells {
		cells[i] = fdother.RawCell{Width: 24, Height: 20, Pixels: bytes.Repeat([]byte{byte(32 + i)}, 24*20)}
	}
	g := &Game{nativeMapAssets: assets, m: field, st: state, sel: u,
		camp:                 campaign.NewRunner(&campaign.Campaign{Start: "battle", Nodes: map[string]*campaign.Node{"battle": {Type: "battle"}}}),
		nativeActionCellsRaw: cells, ring: true, actionOverlayPhase: actionOverlayOpen,
		actionOverlayFrame: 2, nativeActionOverlayOpenFrameVariant: true,
		actionOverlayDrawn: true, actionOverlayShotHold: true, nativeRNGState: 177,
		nativeMapFrozenNow: func() time.Time { return time.Unix(500, 0) },
	}
	r := parityReplay{t: t, g: g, out: t.TempDir()}
	before, _ := json.Marshal(state)
	r.frame("move")
	if g.actionOverlayPhase != actionOverlayOpen || g.actionOverlayFrame != 2 ||
		!g.nativeActionOverlayOpenFrameVariant || !g.actionOverlayDrawn || !g.actionOverlayShotHold ||
		g.nativeRNGState != 177 || g.sel != u || !g.ring {
		t.Fatal("candidate enumeration changed the lifecycle, input or RNG owner")
	}
	after, _ := json.Marshal(state)
	if !bytes.Equal(before, after) {
		t.Fatal("candidate enumeration changed typed battle state")
	}
	groups := append([]parityActionOverlayCandidate(nil), r.actionOverlayCandidatesProvenance...)
	if len(groups) != 6 {
		t.Fatalf("candidate groups=%+v", groups)
	}
	first := map[string][32]byte{}
	files, err := filepath.Glob(filepath.Join(r.out, "remake-*-p*.png"))
	if err != nil || len(files) != 48 {
		t.Fatalf("full frame candidates=%d err=%v", len(files), err)
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		first[path] = sha256.Sum256(raw)
	}
	r.frame("move")
	if !reflect.DeepEqual(groups, r.actionOverlayCandidatesProvenance) {
		t.Fatal("candidate provenance changed on repeat")
	}
	for path, sum := range first {
		raw, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(raw) != sum {
			t.Fatal("full candidate frame was not reproducible")
		}
	}
}
