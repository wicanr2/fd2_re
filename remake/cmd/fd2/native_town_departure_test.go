package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestNativeTownDeparturePreflightAndPresentBoundary(t *testing.T) {
	assets, _, _ := completeNativeMapFrameFixture(t)
	variant := 2
	c := &campaign.Campaign{Start: "prep", Nodes: map[string]*campaign.Node{
		"prep": {Type: "preparation", Cancel: "town"},
		"town": {Type: "town", NativeTownVariant: &variant},
	}}
	g := &Game{m: &MapData{}, camp: campaign.NewRunner(c), nativeMapAssets: assets, prepPromptSource: bytes.Repeat([]byte{7}, 64000)}
	for _, bad := range []string{"source", "variant", "palette", "busy"} {
		t.Run(bad, func(t *testing.T) {
			test := *g
			switch bad {
			case "source":
				test.prepPromptSource = test.prepPromptSource[:63999]
			case "variant":
				variant = 3
				defer func() { variant = 2 }()
			case "palette":
				test.nativeMapAssets = nil
			case "busy":
				test.nativePaletteRamp = &nativePaletteRampJob{}
			}
			if test.startNativeTownDeparture(nil) == nil || test.nativeTownDeparture != nil || test.camp.NodeID() != "prep" {
				t.Fatal("invalid input published an owner or advanced the campaign")
			}
		})
	}
	advanced := 0
	if err := g.startNativeTownDeparture(func() { advanced++ }); err != nil {
		t.Fatal(err)
	}
	for step := 0; step < 11; step++ {
		g.stepNativeTownDeparture()
		if g.nativeTownDeparture.step != step || advanced != 0 {
			t.Fatal("advanced without Draw")
		}
		g.Draw(ebiten.NewImage(640, 400))
		g.stepNativeTownDeparture()
	}
	if advanced != 1 || !nativeDACIsBlack(g.nativeMapDAC) || !g.nativeTownLoadPending {
		t.Fatal("missing black boundary or one-time continuation")
	}
}

type townDepartureOracleRow struct {
	EAX     string
	EIP     string
	Step    uint64
	File    string
	Stack   []string
	Runtime struct {
		DAC string `json:"palette_dac6_hex"`
	} `json:"map_runtime"`
}

func townDepartureReadPNG(t *testing.T, path string) *image.Paletted {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := im.(*image.Paletted)
	if !ok || p.Bounds() != image.Rect(0, 0, 320, 200) || len(p.Pix) != 64000 {
		t.Fatal("original indexed PNG shape")
	}
	return p
}

func townDepartureRows(t *testing.T, path string) []townDepartureOracleRow {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows []townDepartureOracleRow
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var row townDepartureOracleRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	return rows
}

func townDepartureDAC(t *testing.T, row townDepartureOracleRow) []byte {
	t.Helper()
	dac, err := hex.DecodeString(row.Runtime.DAC)
	if err != nil || len(dac) != 768 {
		t.Fatal("original DAC", err)
	}
	return dac
}

func townDepartureCompare(t *testing.T, pixels, expected, dac, expectedDAC []byte) map[string]int {
	t.Helper()
	p, err := fdother.VGAPaletteFromDAC(dac)
	if err != nil {
		t.Fatal(err)
	}
	e, err := fdother.VGAPaletteFromDAC(expectedDAC)
	if err != nil {
		t.Fatal(err)
	}
	diff := map[string]int{"indexed": 0, "rgb": 0, "palette": 0}
	for i := range pixels {
		if pixels[i] != expected[i] {
			diff["indexed"]++
		}
		if p[pixels[i]] != e[expected[i]] {
			diff["rgb"]++
		}
	}
	for i := range p {
		if p[i] != e[i] {
			diff["palette"]++
		}
	}
	return diff
}

// Both runs use the same registered 60-key plan and fixed chapter slot.
// The pure consumer uses the actual saved caller pixels. The normal owner
// starts from LOAD and preserves its own snapshot without PNG injection.
func TestNativeTownDepartureFromOracle(t *testing.T) {
	run, sourceRun, slot, out := os.Getenv("FD2_TOWN_DEPARTURE_ORIGINAL"), os.Getenv("FD2_TOWN_DEPARTURE_SOURCE"), os.Getenv("FD2_TOWN_DEPARTURE_SLOT"), os.Getenv("FD2_TOWN_DEPARTURE_OUT")
	if run == "" || sourceRun == "" || slot == "" || out == "" {
		t.Skip("需要#39固定原版過場與source收據")
	}
	stored, err := os.ReadFile(slot)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(stored)) != "85b080cb71d69add3cf404e881836e511a546876873a8b43e8dc4d6a77e0f348" {
		t.Fatal("fixed slot", err)
	}
	sourceRows := townDepartureRows(t, filepath.Join(sourceRun, "frames/frames.jsonl"))
	if len(sourceRows) != 1 || sourceRows[0].EIP != "0x2D1B9" {
		t.Fatal("saved caller boundary")
	}
	source := townDepartureReadPNG(t, filepath.Join(sourceRun, "frames", sourceRows[0].File))
	frames := townDepartureRows(t, filepath.Join(run, "frames/frames.jsonl"))
	if len(frames) != 11 {
		t.Fatal("original deduplicated frames", len(frames))
	}
	trace := townDepartureRows(t, filepath.Join(run, "eip-trace.jsonl"))
	var townDacs, mapDacs [][]byte
	for _, row := range trace {
		if row.EIP == "0x2D25A" {
			townDacs = append(townDacs, townDepartureDAC(t, row))
		}
		if row.EIP == "0x1F544" && len(row.Stack) > 4 && row.Stack[4] == "0x2066E" {
			mapDacs = append(mapDacs, townDepartureDAC(t, row))
		}
	}
	if len(townDacs) != 10 || len(mapDacs) != 65 {
		t.Fatal("normal caller DAC counts", len(townDacs), len(mapDacs))
	}
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", canonicalCampaignReference)
	t.Setenv("FD2_NATIVE_SAVE", slot)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := loadGame()
	if g.loadErr != "" || !g.confirmTitleLoadSlot(0) {
		t.Fatal("normal LOAD", g.loadErr)
	}
	baseline := g.nativeMapAssets.PaletteDAC
	report := map[string]any{"schema_version": 1, "kind": "fd2_town_departure_validation", "scope": "limited RUNTIME-E1; normal LOAD owner, near-state clock; no new PLAYER-E2", "source_sha256": fmt.Sprintf("%x", sha256.Sum256(source.Pix)), "pure": []any{}, "normal_owner": []any{}, "map": []any{}}
	for k := 1; k <= 10; k++ {
		pixels, err := campaign.ComposeNativeTownDepartureFrame(source.Pix, 2, 2, k)
		if err != nil {
			t.Fatal(err)
		}
		original := townDepartureReadPNG(t, filepath.Join(run, "frames", frames[k-1].File))
		dac := make([]byte, 768)
		if err := fdother.ApplyVGAPaletteSubtraction(dac, baseline, 0, 255, 4*k); err != nil {
			t.Fatal(err)
		}
		diff := townDepartureCompare(t, pixels, original.Pix, dac, townDacs[k-1])
		report["pure"] = append(report["pure"].([]any), diff)
		if diff["indexed"] != 0 || diff["rgb"] != 0 || diff["palette"] != 0 {
			t.Fatalf("pure step%d: %v", k, diff)
		}
	}
	r := &parityReplay{t: t, g: g, battle: "battle_ch07"}
	r.settleTown()
	for g.campSel != 2 {
		if !g.moveNativeTownSelection(1) {
			t.Fatal("town cursor")
		}
	}
	// 固定原版最後一次2D010 EAX，來源是選幀consumer，不由圖片反推。
	pulseRows := townDepartureRows(t, filepath.Join(filepath.Dir(run), "town-departure-pulse-r1", "eip-trace.jsonl"))
	selectedFrame := -1
	for _, row := range pulseRows {
		if row.EIP == "0x2D010" {
			value, err := strconv.ParseUint(strings.TrimPrefix(row.EAX, "0x"), 16, 32)
			if err != nil || value > 2 {
				t.Fatal("original town selected frame", row.EAX)
			}
			selectedFrame = int(value)
		}
		if row.EIP == "0x2D093" {
			break
		}
	}
	if selectedFrame != 1 {
		t.Fatal("fixed original town selected frame", selectedFrame)
	}
	g.nativeTownUIPulse = selectedFrame
	report["same_state_sync"] = "原版最後2D010 EAX=1同步城鎮選幀；不匯入圖片，不宣稱跨選單wall-clock一致。"
	r.enterTownOption(2)
	if !pump(t, g, 600, func() bool { return !g.nativeClassUIBlocksInput() }) || !g.handleNativePreparationInput(nativePreparationInput{enter: true}) {
		t.Fatal("normal YES")
	}
	if !pump(t, g, 600, func() bool { return g.nativeTownDeparture != nil }) {
		t.Fatal("normal departure owner", g.loadErr)
	}
	caller := append([]byte(nil), g.prepPromptSource...)
	callerPalette, err := fdother.VGAPaletteFromDAC(g.nativeMapAssets.PaletteDAC)
	if err != nil {
		t.Fatal(err)
	}
	callerImage := image.NewPaletted(image.Rect(0, 0, 320, 200), callerPalette)
	copy(callerImage.Pix, caller)
	callerFile, err := os.Create(out + "-caller.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(callerFile, callerImage); err != nil {
		t.Fatal(err)
	}
	if err := callerFile.Close(); err != nil {
		t.Fatal(err)
	}
	report["normal_source_sha256"] = fmt.Sprintf("%x", sha256.Sum256(caller))
	if os.Getenv("FD2_TOWN_DEPARTURE_GPU") == "1" {
		seenOut, seenIn := [11]bool{}, [65]bool{}
		probe := &townDepartureGPUProbe{g: g, out: out, captured: map[string]bool{}}
		probe.observe = func() {
			if j := g.nativeTownDeparture; j != nil {
				if seenOut[j.step] {
					return
				}
				seenOut[j.step] = true
				k := j.step
				if k < 10 {
					original := townDepartureReadPNG(t, filepath.Join(run, "frames", frames[k].File))
					report["normal_owner"] = append(report["normal_owner"].([]any), townDepartureCompare(t, j.frames[k].indexed, original.Pix, j.frames[k].dac, townDacs[k]))
				}
			} else if j := g.nativePaletteRamp; j != nil {
				if seenIn[j.step] {
					return
				}
				seenIn[j.step] = true
				original := townDepartureReadPNG(t, filepath.Join(run, "frames", frames[10].File))
				report["map"] = append(report["map"].([]any), townDepartureCompare(t, j.vga, original.Pix, j.dac, mapDacs[j.step]))
			}
		}
		if err := ebiten.RunGame(probe); err != nil {
			t.Fatal(err)
		}
		if probe.err != nil {
			t.Fatal(probe.err)
		}
		for _, v := range seenOut {
			if !v {
				t.Fatal("GPU town present missing")
			}
		}
		for _, v := range seenIn {
			if !v {
				t.Fatal("GPU map present missing")
			}
		}
		report["gpu"] = map[string]any{"town_presents": 11, "map_presents": 65, "rgba_pixels_per_present": 256000, "diff_pixels": 0}
	} else {
		screen := ebiten.NewImage(640, 400)
		for k := 0; k < 11; k++ {
			j := g.nativeTownDeparture
			if j == nil || j.step != k {
				t.Fatal("normal step", k)
			}
			g.stepNativeTownDeparture()
			if j.step != k || g.camp.Node().Type != "preparation" {
				t.Fatal("advanced before present")
			}
			g.handleNativePreparationInput(nativePreparationInput{enter: true})
			if g.nativeTownDeparture != j {
				t.Fatal("duplicate YES changed owner")
			}
			if k < 10 {
				original := townDepartureReadPNG(t, filepath.Join(run, "frames", frames[k].File))
				report["normal_owner"] = append(report["normal_owner"].([]any), townDepartureCompare(t, j.frames[k].indexed, original.Pix, j.frames[k].dac, townDacs[k]))
			}
			g.Draw(screen)
			g.stepNativeTownDeparture()
		}
		if g.nativeTownDeparture != nil || g.nativeTownLoadPending || g.nativePaletteRamp == nil || len(g.dialog) != 0 || g.camp.NodeID() != "story_ch07" {
			t.Fatal("LOADCH handoff", g.loadErr, g.camp.NodeID())
		}
		mapOriginal := townDepartureReadPNG(t, filepath.Join(run, "frames", frames[10].File))
		for k := 0; k < 65; k++ {
			j := g.nativePaletteRamp
			if j == nil || j.step != k || len(g.dialog) != 0 {
				t.Fatal("map fade caller advanced early", k)
			}
			g.stepNativePaletteRamp()
			if j.step != k {
				t.Fatal("map advanced without Draw")
			}
			diff := townDepartureCompare(t, j.vga, mapOriginal.Pix, j.dac, mapDacs[k])
			report["map"] = append(report["map"].([]any), diff)
			g.Draw(screen)
			g.stepNativePaletteRamp()
		}

	}
	if g.nativePaletteRamp != nil || !bytes.Equal(g.nativeMapDAC, g.nativeMapAssets.PaletteDAC) {
		t.Fatal("final map caller")
	}
	report["final_node"] = g.camp.NodeID()
	report["final_dialog_count"] = len(g.dialog)
	if os.Getenv("FD2_TOWN_DEPARTURE_GPU") != "1" {
		driveStory(t, g, newJourneyTrace(), func() bool { return g.camp.NodeID() == "battle_ch07" })
		if !pump(t, g, journeyStoryFrames, func() bool {
			if r.playerHasControl() {
				return true
			}
			if len(g.dialog) > 0 && storyEnterReady(g) {
				g.handleBattleEventDialogueInput(true)
			}
			return false
		}) {
			t.Fatal("normal player control after town departure", g.loadErr)
		}
		report["post_story_node"] = g.camp.NodeID()
		report["player_control"] = r.playerHasControl()
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(b, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"normal_owner", "map"} {
		for i, row := range report[kind].([]any) {
			diff := row.(map[string]int)
			if diff["rgb"] > 640 || diff["palette"] != 0 {
				t.Errorf("%s step%d: %v", kind, i, diff)
			}
		}
	}
}

// GPU validation owns the same complete Game and its formal Update/Draw.
type townDepartureGPUProbe struct {
	g        *Game
	out      string
	captured map[string]bool
	observe  func()
	updates  int
	err      error
}

func (p *townDepartureGPUProbe) Layout(_, _ int) (int, int) { return 640, 400 }
func (p *townDepartureGPUProbe) Update() error {
	p.updates++
	if p.err != nil || p.g.nativeTownDeparture == nil && p.g.nativePaletteRamp == nil {
		return ebiten.Termination
	}
	if p.updates > 3000 {
		return fmt.Errorf("town GPU exceeded bound")
	}
	return p.g.Update()
}
func (p *townDepartureGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil {
		return
	}
	var pixels []byte
	var dac []byte
	if j := p.g.nativeTownDeparture; j != nil {
		pixels = j.frames[j.step].indexed
		dac = j.frames[j.step].dac
	} else if j := p.g.nativePaletteRamp; j != nil {
		pixels = j.vga
		dac = j.dac
	} else {
		return
	}
	p.observe()
	p.g.Draw(screen)
	palette, err := fdother.VGAPaletteFromDAC(dac)
	if err != nil {
		p.err = err
		return
	}
	rgba := make([]byte, 640*400*4)
	screen.ReadPixels(rgba)
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			r, g, b, a := palette[pixels[(y/2)*320+x/2]].RGBA()
			i := (y*640 + x) * 4
			if rgba[i] != byte(r>>8) || rgba[i+1] != byte(g>>8) || rgba[i+2] != byte(b>>8) || rgba[i+3] != byte(a>>8) {
				p.err = fmt.Errorf("town GPU RGB mismatch x%d y%d", x, y)
				return
			}
		}
	}
	name := ""
	if j := p.g.nativeTownDeparture; j != nil && (j.step == 0 || j.step == 9) {
		name = fmt.Sprintf("town-%02d", j.step+1)
	}
	if j := p.g.nativePaletteRamp; j != nil && j.step == 64 {
		name = "map-final"
	}
	if name != "" && !p.captured[name] {
		im := image.NewNRGBA(image.Rect(0, 0, 640, 400))
		copy(im.Pix, rgba)
		f, err := os.Create(p.out + "-" + name + ".png")
		if err != nil {
			p.err = err
			return
		}
		if err := png.Encode(f, im); err != nil {
			p.err = err
		}
		if err := f.Close(); err != nil {
			p.err = err
		}
		p.captured[name] = true
	}
}
