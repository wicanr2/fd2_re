package ending

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func TestIndexedCompositorCopiesBlitsAndClampsPalette(t *testing.T) {
	c := NewIndexedCompositor()
	if err := c.Blit(fdother.Frame{X: 1, Y: 1, Width: 2, Height: 1, Pixels: []byte{2, 0, 1, 0, 1, 9}}, c.Offscreen, Width); err != nil {
		t.Fatal(err)
	}
	if err := c.CopyToVGA(c.Offscreen); err != nil {
		t.Fatal(err)
	}
	if c.VGA[Width+1] != 9 || c.VGA[Width+2] != 9 {
		t.Fatalf("blit/copy=%v", c.VGA[Width:Width+4])
	}
	c.Baseline[3], c.Baseline[4], c.Baseline[5] = 2, 62, 63
	c.baselineKnown = true
	if err := c.PaletteDelta(1, 1, 4); err != nil {
		t.Fatal(err)
	}
	if got := c.Palette[3:6]; got[0] != 6 || got[1] != 63 || got[2] != 63 {
		t.Fatalf("palette=%v", got)
	}
}

func TestComposite40UsesNativeViewportAndOffsets(t *testing.T) {
	c := NewIndexedCompositor()
	c.Offscreen[0] = 3
	frames := make([]fdother.Frame, 9)
	for i := 1; i < 9; i++ {
		frames[i] = fdother.Frame{X: 0, Y: 0, Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, byte(i)}}
	}
	if err := c.Composite40(frames, 0); err != nil {
		t.Fatal(err)
	}
	if c.VGA[0] != 3 || c.Work[290] != 1 || c.Work[80] != 5 {
		t.Fatalf("viewport=%d primary=%d secondary=%d", c.VGA[0], c.Work[290], c.Work[80])
	}
}

func TestComposite200UsesBaselinePalette(t *testing.T) {
	c := NewIndexedCompositor()
	c.Offscreen[1] = 3
	c.Baseline[0] = 1
	frames := make([]fdother.Frame, 9)
	for i := 1; i < 9; i++ {
		frames[i] = fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, byte(i)}}
	}
	if err := c.Composite200(frames, 136); err != nil {
		t.Fatal(err)
	}
	if c.VGA[1] != 3 || c.Palette[0] != 0 {
		t.Fatalf("vga=%d palette=%d", c.VGA[1], c.Palette[0])
	}
}

func TestRecoveredPrefixStopsAtNativeOnlyGate(t *testing.T) {
	frame, transparent := 0, -1
	c := NewIndexedCompositor()
	timeline := Timeline{Segments: []Segment{
		{Op: "blit_frame", Source: "test", Frame: &frame, Target: "offscreen", Stride: Width, Transparent: &transparent},
		{Op: "copy_buffer", Source: "test", Bytes: Bytes, From: "offscreen", To: "vga"},
		{Op: "native_call_opaque", Source: "gate"},
	}}
	frames := []fdother.Frame{{X: 0, Y: 0, Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 7}}}
	stopped, err := c.RunRecoveredPrefix(timeline, frames)
	if err == nil || stopped != 2 || c.VGA[0] != 7 {
		t.Fatalf("stopped=%d err=%v pixel=%d", stopped, err, c.VGA[0])
	}
}

func TestPresentANIReplacesIndexedVGAAndPalette(t *testing.T) {
	c := NewIndexedCompositor()
	frame, palette := make([]byte, Bytes), make([]byte, 768)
	frame[7], palette[9] = 42, 63
	if err := c.PresentANI(frame, palette); err != nil {
		t.Fatal(err)
	}
	if c.VGA[7] != 42 || c.Palette[9] != 63 {
		t.Fatalf("ANI state not presented")
	}
}

func TestRGBAConvertsSixBitDAC(t *testing.T) {
	c := NewIndexedCompositor()
	c.VGA[0] = 1
	c.Palette[3], c.Palette[4], c.Palette[5] = 63, 32, 0
	p := c.RGBA().Pix
	if p[0] != 255 || p[1] != 130 || p[2] != 0 || p[3] != 255 {
		t.Fatalf("rgba=%v", p[:4])
	}
}

func TestPresentANIFrameKeepsFramePalettePair(t *testing.T) {
	c := NewIndexedCompositor()
	clip := &afm.Clip{IndexedFrames: [][]byte{make([]byte, Bytes)}, Palettes: [][]byte{make([]byte, 768)}}
	clip.IndexedFrames[0][3], clip.Palettes[0][4] = 11, 12
	if err := c.PresentANIFrame(clip, 0); err != nil || c.VGA[3] != 11 || c.Palette[4] != 12 {
		t.Fatalf("err=%v", err)
	}
}

func TestANIPlayerUsesExactMillisecondCadence(t *testing.T) {
	c := NewIndexedCompositor()
	clip := &afm.Clip{IndexedFrames: [][]byte{make([]byte, Bytes), make([]byte, Bytes)}, Palettes: [][]byte{make([]byte, 768), make([]byte, 768)}}
	clip.IndexedFrames[0][0], clip.IndexedFrames[1][0] = 1, 2
	p := &ANIPlayer{Clip: clip, DelayMs: 100}
	if done, err := p.Advance(c, 0); err != nil || done || c.VGA[0] != 1 {
		t.Fatal("first frame must present immediately")
	}
	if done, err := p.Advance(c, 99); err != nil || done || c.VGA[0] != 1 {
		t.Fatal("advanced too early")
	}
	if done, err := p.Advance(c, 1); err != nil || !done || c.VGA[0] != 2 {
		t.Fatalf("last frame")
	}
}

func TestPlayerPlaysRecoveredPrefixThenBlocksAtPaletteRamp(t *testing.T) {
	frame, transparent := 0, -1
	ani := 2
	timeline := Timeline{Segments: []Segment{
		{Op: "blit_frame", Source: "blit", Frame: &frame, Target: "offscreen", Stride: Width, Transparent: &transparent},
		{Op: "copy_buffer", Source: "copy", Bytes: Bytes, From: "offscreen", To: "vga"},
		{Op: "delay_ms", Source: "wait", Ms: 1000},
		{Op: "ani_play", Source: "ani", ANIResource: &ani, FrameDelayMs: 100, Skippable: boolPtr(false)},
		{Op: "palette_update", Source: "palette", PaletteStart: intPtr(0), PaletteEnd: intPtr(0), PaletteValue: intPtr(2)},
		{Op: "palette_ramp", Source: "ramp", PaletteStart: intPtr(2), PaletteEnd: intPtr(0), PaletteStep: -1, PaletteDelay: 4},
		{Op: "native_text_branch_opaque", Source: "unrecovered"},
	}}
	frames := []fdother.Frame{{X: 0, Y: 0, Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 7}}}
	clip := &afm.Clip{IndexedFrames: [][]byte{make([]byte, Bytes), make([]byte, Bytes)}, Palettes: [][]byte{make([]byte, 768), make([]byte, 768)}}
	clip.IndexedFrames[0][0], clip.IndexedFrames[1][0] = 8, 9
	p, err := NewPlayer(timeline, frames, clip, NewIndexedCompositor())
	if err != nil {
		t.Fatal(err)
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackRunning || p.Compositor.VGA[0] != 7 {
		t.Fatalf("initial state=%s err=%v pixel=%d", state, err, p.Compositor.VGA[0])
	}
	if state, err := p.Advance(999); err != nil || state != PlaybackRunning || p.Compositor.VGA[0] != 7 {
		t.Fatalf("early delay state=%s err=%v pixel=%d", state, err, p.Compositor.VGA[0])
	}
	if state, err := p.Advance(1); err != nil || state != PlaybackRunning || p.Compositor.VGA[0] != 8 {
		t.Fatalf("ANI first state=%s err=%v pixel=%d", state, err, p.Compositor.VGA[0])
	}
	if state, err := p.Advance(100); err != nil || state != PlaybackRunning || p.Compositor.VGA[0] != 9 {
		t.Fatalf("ANI final state=%s err=%v pixel=%d", state, err, p.Compositor.VGA[0])
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackRunning {
		t.Fatalf("ramp begin state=%s err=%v", state, err)
	}
	if state, err := p.Advance(4); err != nil || state != PlaybackRunning {
		t.Fatalf("ramp second state=%s err=%v", state, err)
	}
	if state, err := p.Advance(4); err != nil || state != PlaybackRunning {
		t.Fatalf("ramp third state=%s err=%v", state, err)
	}
	if state, err := p.Advance(4); err != nil || state != PlaybackBlocked || p.Blocked == nil || p.Blocked.Source != "unrecovered" {
		t.Fatalf("blocked state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
}

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }

func TestPlayerRunsRecoveredNativePrefixWithPlayerAssets(t *testing.T) {
	const (
		fdotherPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
		aniPath     = "../../../org_game/炎龍騎士團/FLAME2/ANI.DAT"
	)
	if _, err := os.Stat(fdotherPath); os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	if _, err := os.Stat(aniPath); os.IsNotExist(err) {
		t.Skip("player-provided ANI.DAT is absent")
	}
	timeline, err := LoadTimeline("../../assets/endings/native_2bce5.json")
	if err != nil {
		t.Fatal(err)
	}
	frames, err := fdother.DecodeResource(fdotherPath, timeline.Resource.Index)
	if err != nil {
		t.Fatal(err)
	}
	clip, err := afm.DecodeResource(aniPath, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 111 || len(clip.IndexedFrames) != 26 {
		t.Fatalf("ending resources frames=%d ani=%d", len(frames), len(clip.IndexedFrames))
	}
	p, err := NewPlayer(*timeline, frames, clip, NewIndexedCompositor())
	if err != nil {
		t.Fatal(err)
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackRunning {
		t.Fatalf("setup state=%s err=%v", state, err)
	}
	if state, err := p.Advance(1000); err != nil || state != PlaybackRunning {
		t.Fatalf("first wait state=%s err=%v", state, err)
	}
	if state, err := p.Advance(2500); err != nil || state != PlaybackRunning {
		t.Fatalf("ANI state=%s err=%v", state, err)
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackRunning {
		t.Fatalf("palette setup state=%s err=%v", state, err)
	}
	if state, err := p.Advance(256); err != nil || state != PlaybackRunning {
		t.Fatalf("palette ramp state=%s err=%v", state, err)
	}
	if state, err := p.Advance(2000); err != nil || state != PlaybackBlocked || p.Blocked == nil || p.Blocked.Op != "native_text_branch_opaque" {
		t.Fatalf("prefix gate state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
	if p.Blocked.Source != "0x2be44" || !p.ResumeBlockedDialogue() {
		t.Fatalf("first native text gate did not resume: %#v", p.Blocked)
	}
	if state, err := p.Advance(5000); err != nil || state != PlaybackBlocked ||
		p.Blocked == nil || p.Blocked.Op != "native_text_branch_opaque" ||
		p.Blocked.Source != "0x2bf1c" {
		t.Fatalf("second text gate state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
	if !p.ResumeBlockedDialogue() {
		t.Fatal("second native text gate did not resume")
	}
	if state, err := p.Advance(7000); err != nil || state != PlaybackBlocked ||
		p.Blocked == nil || p.Blocked.Op != "native_post_composite_opaque" ||
		p.Blocked.Source != "0x2c172" {
		t.Fatalf("post-composite gate state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
}

func TestEndingTimelinePreservesVerifiedAudioCueOrder(t *testing.T) {
	timeline, err := LoadTimeline("../../assets/endings/native_2bce5.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline.AudioCues) != 3 {
		t.Fatalf("audio cues=%#v", timeline.AudioCues)
	}
	want := []AudioCue{
		{Source: "0x2c5cf", Track: 4, DriverArg: 0, RuntimeStage: "party_cycle_start", AfterGate: "0x2c548", Trigger: "0x2c405 after phase0 before party cycle"},
		{Source: "0x2c1ac", Track: -1, DriverArg: 1, RuntimeStage: "tail_stop", Trigger: "0x2bce5 after 0x2c405 returns before FDOTHER#60"},
		{Source: "0x2c1f5", Track: 18, DriverArg: 0, RuntimeStage: "tail_start", Trigger: "0x2bce5 after FDOTHER#60 frame before post-montage loop"},
	}
	for i := range want {
		if timeline.AudioCues[i] != want[i] {
			t.Fatalf("audio cue %d=%#v want %#v", i, timeline.AudioCues[i], want[i])
		}
	}
	p, err := NewPlayer(*timeline, nil, nil, NewIndexedCompositor())
	if err != nil {
		t.Fatal(err)
	}
	got := p.VerifiedAudioCues()
	got[0].Track = 99
	if p.VerifiedAudioCues()[0].Track != 4 {
		t.Fatal("audio cue accessor exposed mutable timeline state")
	}
	if cue, ok := p.AudioCueForRuntimeStage("tail_stop"); !ok || cue != want[1] {
		t.Fatalf("tail stop cue=%#v ok=%v", cue, ok)
	}
	if cue, ok := p.AudioCueForRuntimeStage("tail_start"); !ok || cue != want[2] {
		t.Fatalf("tail start cue=%#v ok=%v", cue, ok)
	}
	if cue, ok := p.AudioCueForRuntimeStage("unknown"); ok || cue != (AudioCue{}) {
		t.Fatalf("unknown runtime stage leaked cue=%#v ok=%v", cue, ok)
	}
	p.State = PlaybackBlocked
	p.Blocked = &Segment{Op: "native_finale_montage_opaque", Source: "0x2c548"}
	if cue, ok := p.AudioCueAtBlockedBoundary(); !ok || cue.Track != 4 || cue.DriverArg != 0 || cue.AfterGate != "0x2c548" {
		t.Fatalf("blocked cue=%#v ok=%v", cue, ok)
	}
	p.Blocked.Source = "0x2c172"
	if cue, ok := p.AudioCueAtBlockedBoundary(); ok || cue != (AudioCue{}) {
		t.Fatalf("unrecovered boundary leaked cue=%#v ok=%v", cue, ok)
	}
}

func TestBlockedDialogueSelectsOnlyNativeTextBranch(t *testing.T) {
	p := &Player{State: PlaybackBlocked, Blocked: &Segment{Op: "native_text_branch_opaque", ThenDialogue: []DialogueBlock{{PortraitID: 4}}, ElseDialogue: []DialogueBlock{{PortraitID: 37}}}}
	if blocks, ok := p.BlockedDialogue(26); !ok || len(blocks) != 1 || blocks[0].PortraitID != 4 {
		t.Fatalf("chapter26 blocks=%#v ok=%v", blocks, ok)
	}
	if blocks, ok := p.BlockedDialogue(29); !ok || len(blocks) != 1 || blocks[0].PortraitID != 37 {
		t.Fatalf("final blocks=%#v ok=%v", blocks, ok)
	}
	p.Blocked.Op = "native_composite_loop_opaque"
	if blocks, ok := p.BlockedDialogue(29); ok || blocks != nil {
		t.Fatalf("non-text opaque block leaked dialogue: %#v", blocks)
	}
	if p.ResumeBlockedDialogue() {
		t.Fatal("non-text opaque block resumed")
	}
	p.Blocked.Op = "native_text_branch_opaque"
	if !p.ResumeBlockedDialogue() || p.State != PlaybackRunning || p.Blocked != nil || p.Segment != 1 {
		t.Fatalf("text resume state=%s blocked=%#v segment=%d", p.State, p.Blocked, p.Segment)
	}
}

func TestPlayerCanResumeTwoRecoveredNativeTextGates(t *testing.T) {
	timeline := Timeline{Segments: []Segment{
		{Op: "native_text_branch_opaque", Source: "first",
			ElseDialogue: []DialogueBlock{{PortraitID: 37}}},
		{Op: "native_text_branch_opaque", Source: "second",
			ElseDialogue: []DialogueBlock{{PortraitID: 45}}},
		{Op: "still_opaque", Source: "stop"},
	}}
	p, err := NewPlayer(timeline, nil, nil, NewIndexedCompositor())
	if err != nil {
		t.Fatal(err)
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackBlocked ||
		p.Blocked == nil || p.Blocked.Source != "first" {
		t.Fatalf("first gate state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
	if !p.ResumeBlockedDialogue() {
		t.Fatal("first gate did not resume")
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackBlocked ||
		p.Blocked == nil || p.Blocked.Source != "second" {
		t.Fatalf("second gate state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
	if !p.ResumeBlockedDialogue() {
		t.Fatal("second gate did not resume")
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackBlocked ||
		p.Blocked == nil || p.Blocked.Source != "stop" {
		t.Fatalf("post-dialogue gate state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
	if p.ResumeBlockedDialogue() {
		t.Fatal("unrecovered post-dialogue gate was skipped")
	}
}

func TestPlayerExpandsRepeatedPaletteRampBeforeNextGate(t *testing.T) {
	timeline := Timeline{Segments: []Segment{{Op: "palette_ramp_repeat", Source: "repeat", PaletteStart: intPtr(1), PaletteEnd: intPtr(0), PaletteStep: -1, PaletteDelay: 4, Repeat: 2, TailDelay: 3}, {Op: "native_text_branch_opaque", Source: "gate"}}}
	compositor := NewIndexedCompositor()
	compositor.baselineKnown = true
	p, err := NewPlayer(timeline, nil, nil, compositor)
	if err != nil {
		t.Fatal(err)
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackRunning {
		t.Fatal(state, err)
	}
	// (1,0)×2 at 4ms plus two 3ms tails.
	if state, err := p.Advance(22); err != nil || state != PlaybackBlocked || p.Blocked == nil || p.Blocked.Source != "gate" {
		t.Fatalf("state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
}

func TestPlayerExpandsTimedFrameSequenceBeforeNextGate(t *testing.T) {
	first, last, transparent := 0, 1, -1
	frames := []fdother.Frame{{X: 0, Y: 0, Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 5}}, {X: 0, Y: 0, Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 6}}}
	timeline := Timeline{Segments: []Segment{{Op: "blit_frame_sequence", Source: "sequence", FirstFrame: &first, LastFrame: &last, Target: "vga", Stride: Width, Transparent: &transparent, PaletteDelay: 20}, {Op: "native_text_branch_opaque", Source: "gate"}}}
	p, err := NewPlayer(timeline, frames, nil, NewIndexedCompositor())
	if err != nil {
		t.Fatal(err)
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackRunning || p.Compositor.VGA[0] != 5 {
		t.Fatalf("first frame state=%s err=%v pixel=%d", state, err, p.Compositor.VGA[0])
	}
	if state, err := p.Advance(40); err != nil || state != PlaybackBlocked || p.Compositor.VGA[0] != 6 || p.Blocked == nil {
		t.Fatalf("sequence state=%s err=%v pixel=%d blocked=%#v", state, err, p.Compositor.VGA[0], p.Blocked)
	}
}

func TestPlayerRunsRecoveredComposite40BeforeTextGate(t *testing.T) {
	frames := make([]fdother.Frame, 9)
	for i := 1; i < 9; i++ {
		frames[i] = fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, byte(i)}}
	}
	p, err := NewPlayer(Timeline{Segments: []Segment{{Op: "native_composite_loop_opaque", Source: "0x2bf60"}, {Op: "native_text_branch_opaque", Source: "gate"}}}, frames, nil, NewIndexedCompositor())
	if err != nil {
		t.Fatal(err)
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackRunning {
		t.Fatal(state, err)
	}
	if state, err := p.Advance(780); err != nil || state != PlaybackBlocked || p.Blocked == nil || p.Blocked.Source != "gate" {
		t.Fatalf("state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
}

func TestPlayerRunsRecoveredComposite200ThenStopsAtPostCompositeGate(t *testing.T) {
	frames := make([]fdother.Frame, 9)
	for i := 1; i < 9; i++ {
		frames[i] = fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, byte(i)}}
	}
	c := NewIndexedCompositor()
	c.Baseline[0] = 3
	p, err := NewPlayer(Timeline{Segments: []Segment{
		{Op: "native_composite_loop_baseline", Source: "0x2c0c5"},
		{Op: "native_post_composite_opaque", Source: "0x2c172"},
	}}, frames, nil, c)
	if err != nil {
		t.Fatal(err)
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackRunning {
		t.Fatalf("initial state=%s err=%v", state, err)
	}
	// The initial pass is immediate; the remaining 199 passes take 20ms each.
	if state, err := p.Advance(3980); err != nil || state != PlaybackBlocked || p.Blocked == nil || p.Blocked.Source != "0x2c172" {
		t.Fatalf("state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
	if c.Palette[0] != 2 {
		t.Fatalf("final baseline delta palette=%d, want 2", c.Palette[0])
	}
}

func TestPlayerPhase0BridgeNeedsNativePaletteAndStopsAtMontageGate(t *testing.T) {
	text, err := os.ReadFile("../../../extracted/raw/FDTXT/FDTXT_031.bin")
	if os.IsNotExist(err) {
		t.Skip("player-provided finale text is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	font, err := os.ReadFile("../../../extracted/raw/FDOTHER/FDOTHER_004.bin")
	if os.IsNotExist(err) {
		t.Skip("player-provided native font is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	phase, err := LoadFinalePhase("../../assets/endings/native_2c405.json")
	if err != nil {
		t.Fatal(err)
	}
	c := NewIndexedCompositor()
	var palette [768]byte
	palette[0] = 50
	if err := c.PresentANI(make([]byte, Bytes), palette[:]); err != nil {
		t.Fatal(err)
	}
	p, err := NewPlayer(Timeline{Segments: []Segment{{Op: "native_post_composite_opaque", Source: "0x2c172"}}}, nil, nil, c)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.EnableRecoveredPhase0(*phase, Phase0Assets{TextResource: text, FontResource: font}); err != nil {
		t.Fatal(err)
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackRunning || p.phase0 == nil || c.Palette[0] != 10 {
		t.Fatalf("initial phase bridge state=%s err=%v phase=%#v palette=%d", state, err, p.phase0, c.Palette[0])
	}
	if state, err := p.Advance(500); err != nil || state != PlaybackBlocked || p.Blocked == nil || p.Blocked.Op != "native_finale_montage_opaque" || p.Blocked.Source != "0x2c548" {
		t.Fatalf("phase bridge state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
}

func TestPlayerPhase0BridgeRefusesUnprovenPalette(t *testing.T) {
	phase, err := LoadFinalePhase("../../assets/endings/native_2c405.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewPlayer(Timeline{Segments: []Segment{{Op: "native_post_composite_opaque", Source: "0x2c172"}}}, nil, nil, NewIndexedCompositor())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.EnableRecoveredPhase0(*phase, Phase0Assets{TextResource: []byte{1}, FontResource: []byte{1}}); err != nil {
		t.Fatal(err)
	}
	if state, err := p.Advance(0); err != nil || state != PlaybackBlocked || p.Blocked == nil || p.Blocked.Source != "0x2c172" {
		t.Fatalf("state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
}

func TestRenderFigureFadePassRestoresBackdropThenShiftsSecondaryFrame(t *testing.T) {
	c := NewIndexedCompositor()
	c.Baseline[0] = 50
	work, restore := make([]byte, Width*Height*2), make([]byte, Bytes)
	for i := range restore {
		restore[i] = 1
	}
	f := figani.Frame{X: 2, Y: 3, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}
	if err := RenderFigureFadePass(c, work, restore, transparentTAI003Fixture(), f, FigureFadePass{Stage: 8, SourceOffset: 80, PaletteDelta: 48}); err != nil {
		t.Fatal(err)
	}
	if c.VGA[0] != 1 || c.VGA[3*Width+82] != 9 || c.Palette[0] != 2 {
		t.Fatalf("vga/palette=%d/%d/%d", c.VGA[0], c.VGA[3*Width+82], c.Palette[0])
	}
}

func TestRenderMirrorFigureFadePassUsesRightViewportAndArg4Gate(t *testing.T) {
	c := NewIndexedCompositor()
	work := make([]byte, Width*Height*2)
	for y := 0; y < Height; y++ {
		work[y*Width*2+Width] = 1
	}
	primary := figani.Frame{X: 100, Y: 3, Width: 1, Height: 1, Pixels: []byte{9}, Mask: []byte{1}}
	secondary := figani.Frame{X: 120, Y: 5, Width: 1, Height: 1, Pixels: []byte{7}, Mask: []byte{1}}
	pass := MirrorFigureFadePass{Stage: 8, PrimarySourceOffset: 0xf0, PaletteDelta: 48, DrawSecondary: true, DrawPlatform: true}
	if err := RenderMirrorFigureFadePass(c, work, transparentTAI003Fixture(), primary, secondary, pass); err != nil {
		t.Fatal(err)
	}
	if c.VGA[0] != 1 || c.VGA[3*Width+20] != 9 || c.VGA[5*Width+120] != 7 || c.Palette[0] != 0 {
		t.Fatalf("vga/palette=%d/%d/%d/%d", c.VGA[0], c.VGA[3*Width+20], c.VGA[5*Width+120], c.Palette[0])
	}
	pass.DrawSecondary, pass.DrawPlatform = false, false
	if err := RenderMirrorFigureFadePass(c, work, fdother.Frame{}, primary, secondary, pass); err != nil {
		t.Fatal(err)
	}
	pass.DrawPlatform = true
	badTAI := transparentTAI003Fixture()
	badTAI.Mask[0] = 255
	if err := RenderMirrorFigureFadePass(c, work, badTAI, primary, secondary, pass); err == nil {
		t.Fatal("mirror platform accepted non-transparent TAI#3")
	}
}

func transparentTAI003Fixture() fdother.Frame {
	return fdother.Frame{Width: 10, Height: 3, Indexed: make([]byte, 30), Mask: make([]byte, 30)}
}
