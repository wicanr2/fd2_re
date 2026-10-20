package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func deathHDTestSet(g *Game) *modernStoryPortraitSet {
	set := &modernStoryPortraitSet{hasHD: true, mapSpritesHD: make(map[int][]image.Image),
		mapOverlays: &modernMapOverlayHDResource{frames: make(map[int]*image.RGBA), native: make(map[int]modernLMIHDDescriptor)}}
	for group := 0; group < fdicon.NativeSourceGroupCount; group++ {
		if !fdicon.NativeSourceGroupHasPixels(group) {
			continue
		}
		set.mapSpritesHD[group] = make([]image.Image, 12)
		picture := image.NewRGBA(image.Rect(0, 0, 144, 144))
		for i := 0; i < len(picture.Pix); i += 4 {
			picture.Pix[i], picture.Pix[i+1], picture.Pix[i+2], picture.Pix[i+3] = 3, 3, 3, 255
		}
		for i := range set.mapSpritesHD[group] {
			set.mapSpritesHD[group][i] = picture
		}
	}
	for index := 68; index <= 79; index++ {
		e := g.nativeMapAssets.CommandHealDigits[index]
		picture := image.NewRGBA(image.Rect(0, 0, e.Width*6, e.Height*6))
		for y := 0; y < e.Height*6; y++ {
			for x := 0; x < e.Width*6; x++ {
				value := e.Pixels[(y/6)*e.Width+x/6]
				if value != 0 {
					picture.Set(x, y, g.nativeMapAssets.Palette[value])
				}
			}
		}
		set.mapOverlays.frames[index] = picture
		set.mapOverlays.native[index] = modernLMIHDDescription(index, e)
	}
	return set
}

func TestModernHDDeathScheduleAndAtomicPreflight(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	native, g := completeNativeCh28PostGame(t), completeNativeCh28PostGame(t)
	g.modernStoryPortraits = deathHDTestSet(g)
	if err := native.startNativeCh28PostPresent(nil); err != nil {
		t.Fatal(err)
	}
	if err := g.startNativeCh28PostPresent(nil); err != nil {
		t.Fatal(err)
	}
	j, baseline := g.nativeCh28PostPresent, native.nativeCh28PostPresent
	if !reflect.DeepEqual(j.workFrames, baseline.workFrames) || !reflect.DeepEqual(j.vgaFrames, baseline.vgaFrames) ||
		!reflect.DeepEqual(j.stateFrames, baseline.stateFrames) || !reflect.DeepEqual(j.waits, baseline.waits) {
		t.Fatal("HD changed the native transaction")
	}
	for i, frame := range j.hdFrames {
		want := 0
		if i >= 13 && i < 19 {
			want = (i - 12) * len(j.hdTargets)
		}
		if i >= 19 && i < 25 {
			want = len(j.hdTargets)
		}
		if len(frame.Overlays) != want || frame.Range != nil {
			t.Fatalf("frame %d effect accumulation/range differs", i)
		}
		if i < 25 && bytes.Count(frame.UI.Pix, []byte{0}) != len(frame.UI.Pix) {
			t.Fatal("HUD was redrawn before steady")
		}
		if i >= 13 && i < 19 && !reflect.DeepEqual(frame.Units, j.hdFrames[12].Units) {
			t.Fatal("A removed the last visible pose")
		}
		if i >= 19 && i < 25 && len(frame.Units) >= len(j.hdFrames[12].Units) {
			t.Fatal("B retained dead units")
		}
	}
	bad := deathHDTestSet(g)
	delete(bad.mapOverlays.frames, 79)
	before, frames, position := g.st, j.hdFrames[0], j.frame
	if err := g.applyTheme(bad); err == nil || g.st != before || g.modernStoryPortraits == bad || j.frame != position || j.hdFrames[0] != frames {
		t.Fatal("failed F2 published partial state or art")
	}
	for _, damage := range []string{"image", "native", "unit"} {
		t.Run(damage, func(t *testing.T) {
			g := completeNativeCh28PostGame(t)
			set := deathHDTestSet(g)
			g.modernStoryPortraits = set
			switch damage {
			case "image":
				delete(set.mapOverlays.frames, 79)
			case "native":
				set.mapOverlays.native[68] = modernLMIHDDescriptor{}
			case "unit":
				set.mapSpritesHD = nil
			}
			before := g.st
			work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
			if err := g.startNativeCh28PostPresent(nil); err == nil || g.nativeCh28PostPresent != nil || g.st != before ||
				!bytes.Equal(work, g.nativeMapWork) || !bytes.Equal(vga, g.nativeMapVGA) {
				t.Fatal("bad HD source mutated the native preflight")
			}
		})
	}
}

func TestModernHDDeathAssetLoader(t *testing.T) {
	root := t.TempDir()
	native := make([]fdother.LMI1Entry, 80)
	asset := modernThemeAsset{Role: "map_overlay_hd_set", Status: "runtime_candidate", Width: 1920, Height: 1200,
		SourceArchive: "FDOTHER.DAT", SourceResource: 5, ConsumerContract: "native_lmi1_rgba_hd6_v1",
		AlphaContract: "coverage", CyclePolicy: "native_lmi1_entries", LayoutPolicy: "native_lmi1_rect_hd6", FrameCount: 12}
	for index := 68; index <= 79; index++ {
		native[index] = fdother.LMI1Entry{Width: 2, Height: 2, Pixels: []byte{0, 5, 0, 0}}
		name := fmt.Sprintf("%d.png", index)
		picture := image.NewRGBA(image.Rect(0, 0, 12, 12))
		picture.Set(7, 1, color.White)
		var out bytes.Buffer
		if err := png.Encode(&out, picture); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), out.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(out.Bytes())
		asset.Files = append(asset.Files, name)
		asset.FrameSHA256 = append(asset.FrameSHA256, hex.EncodeToString(digest[:]))
		asset.NativeLMIFrames = append(asset.NativeLMIFrames, modernLMIHDDescription(index, native[index]))
	}
	set := &modernStoryPortraitSet{}
	if err := loadModernMapOverlayHDWithNative(set, asset, root, native); err != nil || len(set.mapOverlays.frames) != 12 {
		t.Fatalf("valid bank rejected: %v", err)
	}
	for _, kind := range []string{"order", "geometry", "pixels", "path", "hash", "missing", "identity", "contract"} {
		t.Run(kind, func(t *testing.T) {
			bad := asset
			bad.Files = append([]string(nil), asset.Files...)
			bad.FrameSHA256 = append([]string(nil), asset.FrameSHA256...)
			bad.NativeLMIFrames = append([]modernLMIHDDescriptor(nil), asset.NativeLMIFrames...)
			switch kind {
			case "order":
				bad.NativeLMIFrames[11].Index = 68
			case "geometry":
				bad.NativeLMIFrames[11].Width++
			case "pixels":
				bad.NativeLMIFrames[11].PixelsSHA256 = "wrong"
			case "path":
				bad.Files[11] = "../79.png"
			case "hash":
				bad.FrameSHA256[11] = "wrong"
			case "missing":
				bad.Files[11] = "missing.png"
			case "identity":
				bad.SourceResource = 6
			case "contract":
				bad.ConsumerContract = "native_figani_rgba_hd6_v1"
			}
			target := &modernStoryPortraitSet{}
			if err := loadModernMapOverlayHDWithNative(target, bad, root, native); err == nil || target.mapOverlays != nil {
				t.Fatal("bad bank partially admitted")
			}
		})
	}
}

func TestModernHDDeathRetainsAuxiliarySeedWithoutSteadyWriter(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g := completeNativeCh28PostGame(t)
	set := deathHDTestSet(g)
	if err := g.startNativeCh28PostPresent(nil); err != nil {
		t.Fatal(err)
	}
	job := g.nativeCh28PostPresent
	baseline, err := g.prepareModernHDDeathFrames(job, set)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 25; i++ {
		job.hdInputs[i].Frame.ChapterAux = &fdother.NativeChapterAuxSurface{}
	}
	event, err := g.prepareModernHDDeathFrames(job, set)
	if err != nil {
		t.Fatalf("event invoked the steady auxiliary writer: %v", err)
	}
	for i := range event {
		if !bytes.Equal(event[i].Background.Pix, baseline[i].Background.Pix) {
			t.Fatalf("event frame%d replaced its auxiliary seed", i)
		}
	}
	job.hdInputs[25].Frame.ChapterAux = &fdother.NativeChapterAuxSurface{}
	if _, err := g.prepareModernHDDeathFrames(job, set); err == nil {
		t.Fatal("steady compose skipped its declared auxiliary writer")
	}
}

type modernHDDeathProbe struct {
	g                              *Game
	set                            *modernStoryPortraitSet
	job                            *nativeCh28PostPresentJob
	presentation                   *modernHDPresentation
	err                            error
	done                           bool
	steps, draws, finals, faithful int
	seen                           [26]bool
	toggled                        [26]bool
	restore                        bool
	faithfulAt                     int
	out                            string
	records                        []map[string]any
	diagnostic                     bool
}

func (p *modernHDDeathProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *modernHDDeathProbe) Update() error {
	p.steps++
	if p.err != nil || p.done {
		return ebiten.Termination
	}
	if p.steps > 600 {
		p.err = fmt.Errorf("death owner did not finish")
		return ebiten.Termination
	}
	j := p.g.nativeCh28PostPresent
	if j == nil {
		p.err = fmt.Errorf("owner disappeared")
		return ebiten.Termination
	}
	if p.restore {
		if p.faithful <= p.faithfulAt {
			return nil
		}
		p.restore = false
		if err := p.g.applyTheme(p.set); err != nil {
			p.err = err
		}
		return nil
	}
	if p.seen[j.frame] && !p.toggled[j.frame] && (j.frame == 1 || j.frame == 14 || j.frame == 20 || j.frame == 25) {
		before, wait, drawn := p.g.st, j.wait, j.drawn
		if err := p.g.applyTheme(nil); err != nil {
			p.err = err
		}
		if p.g.st != before || j.wait != wait || j.drawn != drawn {
			p.err = fmt.Errorf("F2 advanced native owner")
		}
		p.toggled[j.frame], p.restore = true, true
		p.faithfulAt = p.faithful
		return nil
	}
	p.g.stepNativeCh28PostPresent()
	return nil
}
func (p *modernHDDeathProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.presentation.DrawFinalScreen(screen, offscreen, geometry)
	p.finals++
}
func (p *modernHDDeathProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.done {
		return
	}
	p.draws++
	p.g.Draw(screen)
	if p.g.loadErr != "" {
		p.err = fmt.Errorf("Draw: %s", p.g.loadErr)
		return
	}
	if p.g.modernStoryPortraits == nil {
		p.faithful++
		return
	}
	if !p.g.modernHDWorldVisible || !p.job.drawn {
		p.err = fmt.Errorf("HD death owner not acknowledged")
		return
	}
	i := p.job.frame
	if p.seen[i] {
		return
	}
	p.seen[i] = true
	final := ebiten.NewImage(1920, 1200)
	defer final.Dispose()
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.presentation.DrawFinalScreen(final, screen, geometry)
	pixels := make([]byte, 1920*1200*4)
	final.ReadPixels(pixels)
	if p.diagnostic {
		for y := 0; y < 1200; y++ {
			for x := 0; x < 1920; x++ {
				if i == 25 {
					nearUI := false
					for dy := -1; dy <= 1; dy++ {
						for dx := -1; dx <= 1; dx++ {
							if p.job.hdFrames[i].UI.RGBAAt(x/6+dx, y/6+dy).A != 0 {
								nearUI = true
							}
						}
					}
					if nearUI {
						continue
					}
				}
				want := color.RGBAModel.Convert(p.job.palette[p.job.vgaFrames[i][(y/6)*320+x/6]]).(color.RGBA)
				o := (y*1920 + x) * 4
				if pixels[o] != want.R || pixels[o+1] != want.G || pixels[o+2] != want.B || pixels[o+3] != 255 {
					var covering []modernHDMapUnitDraw
					for _, unit := range p.job.hdFrames[i].Units {
						if image.Pt(x, y).In(unit.Rect) {
							covering = append(covering, unit)
						}
					}
					p.err = fmt.Errorf("diagnostic frame%d at%d,%d got%v want%v covering%+v", i, x, y, pixels[o:o+4], want, covering)
					return
				}
			}
		}
	}
	if p.out != "" {
		name := fmt.Sprintf("frame-%02d.png", i)
		file, err := os.Create(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		err = png.Encode(file, &image.RGBA{Pix: pixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			p.err = err
			return
		}
		raw, err := os.ReadFile(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		p.records = append(p.records, map[string]any{"frame": i, "file": name, "sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "overlays": len(p.job.hdFrames[i].Overlays), "units": len(p.job.hdFrames[i].Units)})
		record := p.records[len(p.records)-1]
		frame := p.job.hdFrames[i]
		record["unit_draws"] = frame.Units
		record["viewport"] = frame.Viewport
		record["foreground_count"], record["terrain_count"] = len(frame.Foreground), len(frame.Terrain)
		var overlays []map[string]any
		for _, overlay := range frame.Overlays {
			overlays = append(overlays, map[string]any{"entry": overlay.Entry, "rect": overlay.Rect})
		}
		record["overlay_draws"] = overlays
		for key, picture := range map[string]*image.RGBA{"background": frame.Background, "ui": frame.UI} {
			path := fmt.Sprintf("%s-%02d.png", key, i)
			file, err := os.Create(filepath.Join(p.out, path))
			if err != nil {
				p.err = err
				return
			}
			err = png.Encode(file, picture)
			closeErr := file.Close()
			if err == nil {
				err = closeErr
			}
			if err != nil {
				p.err = err
				return
			}
			raw, err := os.ReadFile(filepath.Join(p.out, path))
			if err != nil {
				p.err = err
				return
			}
			record[key] = map[string]string{"file": path, "sha256": fmt.Sprintf("%x", sha256.Sum256(raw))}
		}
	}
}

func TestModernHDDeathRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_TEST_HD_DEATH_GPU") != "1" {
		t.Skip("explicit complete Game.Draw probe")
	}
	t.Setenv("FD2_MUTE", "1")
	g := completeNativeCh28PostGame(t)
	// The generic fixture intentionally starts with a black raw DAC. A visible
	// GPU probe must not let that black-out hide a missing HD layer.
	g.nativeMapDAC = nil
	set := deathHDTestSet(g)
	diagnostic := true
	if catalog := os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"); catalog != "" {
		full, err := loadModernStoryPortraitSet(catalog, os.Getenv("FD2_TEST_MODERN_BATTLE_PACK"))
		if err != nil {
			t.Fatal(err)
		}
		lmi, err := fdother.LoadSeparatedLMI1Bank(separatedAssetPath("ui/fdother_005_lmi1_opaque"), 5)
		if err != nil {
			t.Fatal(err)
		}
		g.nativeMapAssets.CommandHealDigits = lmi
		dac, palette, err := loadNativeBattlePalette()
		if err != nil {
			t.Fatal(err)
		}
		g.nativeMapAssets.Palette, g.nativeMapAssets.PaletteDAC = palette, dac
		g.nativeMapDAC = append([]byte(nil), dac...)
		set = &modernStoryPortraitSet{hasHD: true, mapSpritesHD: full.mapSpritesHD, mapOverlays: full.mapOverlays}
		diagnostic = false
	}
	g.modernStoryPortraits = set
	g.baseMapTileset = image.NewRGBA(image.Rect(0, 0, 24, 24))
	p := &modernHDDeathProbe{g: g, set: set, diagnostic: diagnostic, out: os.Getenv("FD2_TEST_HD_DEATH_OUT")}
	continued := 0
	if err := g.startNativeCh28PostPresent(func() { continued++; p.done = true }); err != nil {
		t.Fatal(err)
	}
	p.job = g.nativeCh28PostPresent
	p.presentation = &modernHDPresentation{Game: g}
	before := p.job.frame
	for i := 0; i < 5; i++ {
		g.stepNativeCh28PostPresent()
	}
	if p.job.frame != before {
		t.Fatal("owner advanced without Draw")
	}
	if p.out != "" {
		if err := os.Mkdir(p.out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	ebiten.SetWindowSize(1920, 1200)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || !p.done || continued != 1 || g.nativeCh28PostPresent != nil || p.faithful < 4 || p.finals == 0 {
		t.Fatalf("lifecycle: %v done%v callbacks%d faithful%d", p.err, p.done, continued, p.faithful)
	}
	for i, seen := range p.seen {
		if !seen {
			t.Fatalf("frame%d was never drawn", i)
		}
	}
	if p.out != "" {
		report := map[string]any{"status": "PASS-EVENT-DEATH-FIXTURE-RUNTIME-E1", "diagnostic": diagnostic, "frames": p.records, "hd_frame_count": 26, "faithful_draws": p.faithful, "f2_phases": []string{"pose", "overlay_a", "overlay_b", "steady"}, "continuations": continued, "scope": "既有章節事件fixture；完整Game.Draw與最終1920回讀。不是普通擊殺、正常玩家E2或新原版oracle。"}
		report["targets"] = p.job.hdTargets
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("26 full Game.Draw frames; F2 four phases; once continuation; diagnostic=%v", diagnostic)
}
