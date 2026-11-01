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
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func facilityHDTestSet(t *testing.T) *modernStoryPortraitSet {
	t.Helper()
	catalog, pack := os.Getenv("FD2_TEST_FACILITY_CATALOG"), os.Getenv("FD2_TEST_FACILITY_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit facility candidate or formal pack required")
	}
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil || len(set.facilityScenes) != 6 {
		t.Fatalf("facility preflight: %v", err)
	}
	return set
}

func TestModernHDFacilityRejectsCorruptGroups(t *testing.T) {
	set := facilityHDTestSet(t)
	for _, speaker := range []int{128, 130, 132} {
		if set.portraits[speaker] == nil || set.portraits[speaker].hdImage == nil {
			t.Fatalf("merchant %d absent", speaker)
		}
	}
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_FACILITY_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	var source modernThemeAsset
	for _, a := range catalog.Assets {
		if a.Role == "facility_scene_hd" {
			source = a
			break
		}
	}
	for _, fault := range []string{"hash", "geometry", "source", "metadata", "path", "contract", "duplicate", "alpha"} {
		t.Run(fault, func(t *testing.T) {
			a := source
			root := os.Getenv("FD2_TEST_FACILITY_PACK")
			target := &modernStoryPortraitSet{}
			switch fault {
			case "hash":
				a.SHA256 = "bad"
			case "geometry":
				a.Height--
			case "source":
				a.SourceResource = 13
			case "metadata":
				a.SourceMetadataSHA256 = "bad"
			case "path":
				a.File = "../" + a.File
			case "contract":
				a.ConsumerContract = "guessed"
			case "duplicate":
				target.facilityScenes = map[int]*modernFacilityScene{a.SourceResource: {}}
			case "alpha":
				root = t.TempDir()
				picture := image.NewNRGBA(image.Rect(0, 0, 1920, 1200))
				for i := 3; i < len(picture.Pix); i += 4 {
					picture.Pix[i] = 255
				}
				picture.Pix[3] = 254
				var output bytes.Buffer
				if err := png.Encode(&output, picture); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, a.File), output.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
				d := sha256.Sum256(output.Bytes())
				a.SHA256 = hex.EncodeToString(d[:])
			}
			if err := loadModernFacilityScene(target, a, root); err == nil {
				t.Fatalf("accepted %s", fault)
			}
		})
	}
	for _, fault := range []string{"partial", "duplicate"} {
		t.Run(fault+"-group", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "catalog.json")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			changeModernHDCatalog(t, path, func(d map[string]any) {
				var rows []any
				removed := false
				for _, v := range d["assets"].([]any) {
					role := v.(map[string]any)["role"]
					if role != "facility_scene_hd" && role != "story_portrait_frame" && role != "story_portrait_hd_frame" {
						continue
					}
					if v.(map[string]any)["role"] == "facility_scene_hd" && !removed {
						removed = true
						if fault == "partial" {
							continue
						}
						rows = append(rows, v)
					}
					rows = append(rows, v)
				}
				d["assets"] = rows
			})
			if got, err := loadModernStoryPortraitSet(path, os.Getenv("FD2_TEST_FACILITY_PACK")); err == nil || got != nil {
				t.Fatal("invalid facility group published")
			}
		})
	}
}

func facilityHDTestGame(t *testing.T, set *modernStoryPortraitSet) *Game {
	t.Helper()
	g, _ := newNativeShopRecipientPathGame(t)
	g.font = loadFont()
	attachOfficialLocale(t, g)
	_, glyphs, err := loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeBattleGlyphs = glyphs
	g.nativeTownUI, err = loadNativeTownUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.m = &MapData{TileW: 24, TileH: 24, Cols: 16}
	g.baseMapTileset = image.NewNRGBA(image.Rect(0, 0, 384, 24))
	g.modernStoryPortraits, g.modernThemeCache = set, set
	return g
}

func TestModernHDTownIndependentStylesAndRawCursor(t *testing.T) {
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	variant := 2
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{"town": {Type: "town", NativeTownVariant: &variant}}})
	g.campSel, g.nativeTownUIPulse = 1, 3
	g.partyJoinOrder = []int{7}
	g.partyRoster = map[int]battle.Unit{7: {Portrait: 9, MapSelectorKey: 50, HasMapSelectorKey: true}}
	legacy, ok := g.composeNativeTownFrame()
	if !ok {
		t.Fatal("native town unavailable")
	}
	originalClass := g.nativeClassUI
	before, _ := json.Marshal(g.partyRoster)
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	var hdRuns []modernHDTextRun
	for _, art := range []bool{false, true} {
		for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD} {
			g.modernStoryPortraits = nil
			if art {
				g.modernStoryPortraits = set
			}
			g.interfaceMode, g.modernHDScene = mode, nil
			finish := g.beginModernHDText(screen)
			if !g.drawNativeTown(screen) {
				t.Fatal("town draw failed", art, mode)
			}
			if (g.modernHDScene != nil) != art {
				t.Fatal("town scenery followed interface", art, mode)
			}
			if art {
				cursor := g.modernHDScene.cursor
				if cursor == nil || cursor.group != 50 || cursor.frame != 1 || cursor.rect.Min != image.Pt(84, 888) {
					t.Fatalf("cursor ignored raw first-party key or viewport: %+v", cursor)
				}
			}
			if mode == playerInterfaceHD {
				if len(g.modernHDText.runs) == 0 {
					t.Fatal("HD town has no text", art)
				}
				positions := map[[2]float64]bool{}
				for _, run := range g.modernHDText.runs {
					if !run.inkTop || run.x < 512 || run.y < 344 || !run.clip.In(image.Rect(8, 8, 632, 392)) {
						t.Fatalf("text omitted native viewport offset: %+v", run)
					}
					key := [2]float64{run.x, run.y}
					if positions[key] {
						t.Fatal("mask-only text leaked into displayed runs")
					}
					positions[key] = true
				}
				if !art {
					hdRuns = append([]modernHDTextRun(nil), g.modernHDText.runs...)
				} else if !reflect.DeepEqual(hdRuns, g.modernHDText.runs) {
					t.Fatal("artwork switch changed interface text")
				}
			} else if g.modernHDText != nil {
				t.Fatal("native interface created HD text")
			}
			finish()
			if after, ok := g.composeNativeTownFrame(); !ok || !bytes.Equal(legacy, after) || g.nativeClassUI != originalClass {
				t.Fatal("display changed native assets/frame")
			}
		}
	}
	after, _ := json.Marshal(g.partyRoster)
	if !bytes.Equal(before, after) {
		t.Fatal("display changed persistent party")
	}
	for _, fault := range []string{"raw-key", "missing-record", "missing-hd", "geometry"} {
		t.Run(fault, func(t *testing.T) {
			unit, frames := g.partyRoster[7], set.mapSpritesHD[50]
			defer func() { g.partyRoster[7], set.mapSpritesHD[50] = unit, frames }()
			switch fault {
			case "raw-key":
				bad := unit
				bad.MapSelectorKey = 999
				g.partyRoster[7] = bad
			case "missing-record":
				delete(g.partyRoster, 7)
			case "missing-hd":
				delete(set.mapSpritesHD, 50)
			case "geometry":
				set.mapSpritesHD[50] = []image.Image{nil, image.NewNRGBA(image.Rect(0, 0, 72, 72))}
			}
			if _, err := g.prepareModernTownCursor(variant); err == nil {
				t.Fatal("accepted", fault)
			}
		})
	}
}

type facilityHDProbe struct {
	t            *testing.T
	g            *Game
	presentation *modernHDPresentation
	cases        []func()
	index        int
	drawn        bool
	err          error
	out          string
	checks       []map[string]any
	captured     map[*modernFacilityScene]bool
	headCaptured bool
	captureCases map[int]bool
	verifyScene  func(*facilityHDProbe, []byte, *ebiten.Image) error
}

func (p *facilityHDProbe) Update() error {
	if p.err != nil {
		return ebiten.Termination
	}
	if p.drawn {
		p.index++
		p.drawn = false
	}
	if p.index >= len(p.cases) {
		return ebiten.Termination
	}
	p.cases[p.index]()
	return nil
}
func (p *facilityHDProbe) Layout(int, int) (int, int) { return p.presentation.Layout(0, 0) }
func (p *facilityHDProbe) Draw(screen *ebiten.Image) {
	before, _ := json.Marshal(p.g.partyRoster)
	gold, rng := p.g.gold, p.g.nativeRNGState
	p.presentation.Draw(screen)
	after, _ := json.Marshal(p.g.partyRoster)
	if !bytes.Equal(before, after) || gold != p.g.gold || rng != p.g.nativeRNGState {
		p.err = fmt.Errorf("Draw changed party/gold/RNG")
	}
}
func (p *facilityHDProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.presentation.DrawFinalScreen(screen, offscreen, geometry)
	if p.drawn || p.err != nil {
		return
	}
	final := ebiten.NewImage(1920, 1272)
	defer final.Dispose()
	var m ebiten.GeoM
	m.Scale(3, 3)
	p.presentation.DrawFinalScreen(final, offscreen, m)
	pixels := make([]byte, 1920*1272*4)
	final.ReadPixels(pixels)
	if p.verifyScene != nil {
		if err := p.verifyScene(p, pixels, offscreen); err != nil {
			p.err = err
			return
		}
	} else if p.g.modernStoryPortraits != nil {
		o := p.g.modernHDScene
		if o == nil {
			p.err = fmt.Errorf("case %d lacks HD scene", p.index)
			return
		}
		base := make([]byte, 640*424*4)
		offscreen.ReadPixels(base)
		checked := 0
		for y := 0; y < 1200; y += 11 {
			for x := 0; x < 1920; x += 11 {
				if o.cursor != nil && image.Pt(x, y).In(o.cursor.rect.Intersect(o.cursor.clip)) {
					continue // Cursor pixels have their own source comparison below.
				}
				lx, ly := x/3, y/3+24
				if base[(ly*640+lx)*4+3] != 0 {
					continue
				}
				sx, sy := x-o.dest.X*3+o.source.Min.X, y-o.dest.Y*3+o.source.Min.Y
				if !image.Pt(sx, sy).In(o.source) {
					continue
				}
				want := color.NRGBAModel.Convert(o.scene.image.At(sx, sy)).(color.NRGBA)
				at := ((y+72)*1920 + x) * 4
				got := color.NRGBA{pixels[at], pixels[at+1], pixels[at+2], pixels[at+3]}
				if got != want {
					p.err = fmt.Errorf("case %d HD pixel (%d,%d): %v != %v", p.index, x, y, got, want)
					return
				}
				checked++
			}
		}
		if checked < 1000 {
			p.err = fmt.Errorf("case %d has only %d HD checks", p.index, checked)
			return
		}
		check := map[string]any{"case": p.index, "hd_pixels": checked, "scene": p.g.camp.Node().Type, "variant": p.g.nativeShopVariant}
		if cursor := o.cursor; cursor != nil {
			picture := p.g.modernStoryPortraits.mapSpritesHD[cursor.group][cursor.frame]
			visible := cursor.rect.Intersect(cursor.clip)
			core := 0
			for y := visible.Min.Y; y < visible.Max.Y; y += 2 {
				for x := visible.Min.X; x < visible.Max.X; x += 2 {
					want := color.NRGBAModel.Convert(picture.At(x-cursor.rect.Min.X, y-cursor.rect.Min.Y)).(color.NRGBA)
					if want.A < 250 {
						continue
					}
					background := color.NRGBAModel.Convert(o.scene.image.At(x-o.dest.X*3+o.source.Min.X, y-o.dest.Y*3+o.source.Min.Y)).(color.NRGBA)
					alpha := int(want.A)
					blend := func(source, destination byte) byte {
						return byte((int(source)*alpha + int(destination)*(255-alpha) + 127) / 255)
					}
					want = color.NRGBA{blend(want.R, background.R), blend(want.G, background.G), blend(want.B, background.B), 255}
					at := ((y+72)*1920 + x) * 4
					for channel, value := range []byte{want.R, want.G, want.B, 255} {
						if delta := int(pixels[at+channel]) - int(value); delta < -2 || delta > 2 {
							p.err = fmt.Errorf("case %d town cursor group %d frame %d pixel (%d,%d): %v != %v", p.index, cursor.group, cursor.frame, x, y, pixels[at:at+4], want)
							return
						}
					}
					core++
				}
			}
			if core < 200 {
				p.err = fmt.Errorf("case %d cursor has only %d opaque core checks", p.index, core)
				return
			}
			check["cursor_core_pixels"], check["cursor_group"], check["cursor_frame"] = core, cursor.group, cursor.frame
		}
		p.checks = append(p.checks, check)
	} else {
		if p.g.modernHDScene != nil {
			p.err = fmt.Errorf("case %d original art retained HD scenery", p.index)
			return
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "native_artwork": true, "interface": p.g.interfaceMode})
	}
	if p.g.camp.Node().Type == "town" && p.g.modernHDInterfaceEnabled() {
		if p.g.modernHDText == nil || len(p.g.modernHDText.runs) == 0 {
			p.err = fmt.Errorf("case %d town HD text missing", p.index)
			return
		}
		for _, run := range p.g.modernHDText.runs {
			f := run.font
			if regional := f.hdFonts[p.g.localeID]; regional != nil {
				f = regional
			}
			face, baseline := modernHDTextFace(f, run, p.g.fontScale)
			ink := text.BoundString(face, run.text).Add(image.Pt(int(math.Round(run.x*3)), int(baseline)))
			clip := image.Rectangle{Min: run.clip.Min.Mul(3), Max: run.clip.Max.Mul(3)}
			if !ink.In(clip) {
				p.err = fmt.Errorf("case %d town glyph %s clipped: ink=%v clip=%v", p.index, run.text, ink, clip)
				return
			}
		}
	}
	for _, overlay := range p.g.modernHDPanelPortraits {
		visible := overlay.rect.Intersect(overlay.clip)
		checked := 0
		for y := visible.Min.Y * 3; y < visible.Max.Y*3; y += 3 {
			for x := visible.Min.X * 3; x < visible.Max.X*3; x += 3 {
				sx, sy := x-overlay.rect.Min.X*3, y-overlay.rect.Min.Y*3
				want := color.NRGBAModel.Convert(overlay.portrait.hdImage.At(sx, sy)).(color.NRGBA)
				at := ((y+72)*1920 + x) * 4
				got := color.NRGBA{pixels[at], pixels[at+1], pixels[at+2], pixels[at+3]}
				if got != want {
					p.err = fmt.Errorf("equipment HD portrait pixel (%d,%d): %v != %v", x, y, got, want)
					return
				}
				checked++
			}
		}
		if checked == 0 {
			p.err = fmt.Errorf("only %d portrait core samples", checked)
			return
		}
	}
	// Compare the entire game area against the same state without a toolbar.
	plain := &modernHDPresentation{Game: p.g}
	logical := ebiten.NewImage(640, 400)
	defer logical.Dispose()
	plain.Draw(logical)
	reference := ebiten.NewImage(1920, 1200)
	defer reference.Dispose()
	plain.DrawFinalScreen(reference, logical, m)
	want := make([]byte, 1920*1200*4)
	reference.ReadPixels(want)
	if !bytes.Equal(want, pixels[72*1920*4:]) {
		p.err = fmt.Errorf("case %d toolbar changed game pixels", p.index)
		return
	}
	var scene *modernFacilityScene
	if p.g.modernHDScene != nil {
		scene = p.g.modernHDScene.scene
	}
	if p.captureCases[p.index] || !p.captured[scene] || len(p.g.modernHDPanelPortraits) > 0 && !p.headCaptured || p.index == len(p.cases)-1 {
		file, err := os.Create(filepath.Join(p.out, fmt.Sprintf("scene-%03d.png", p.index)))
		if err == nil {
			err = png.Encode(file, &image.RGBA{Pix: pixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1272)})
			if closeErr := file.Close(); err == nil {
				err = closeErr
			}
		}
		if err != nil {
			p.err = err
			return
		}
		p.captured[scene] = true
		if len(p.g.modernHDPanelPortraits) > 0 {
			p.headCaptured = true
		}
	}
	p.drawn = true
}

func TestModernHDFacilityRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FACILITY_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}, captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	originalFirst := g.partyRoster[g.partyJoinOrder[0]]
	for variant := 0; variant < 3; variant++ {
		for selection := 0; selection < 6; selection++ {
			for pulse := 0; pulse < 4; pulse++ {
				v, s, c := variant, selection, pulse
				p.cases = append(p.cases, func() {
					unit := originalFirst
					unit.MapSelectorKey, unit.HasMapSelectorKey = 50, true
					g.partyRoster[g.partyJoinOrder[0]] = unit
					g.camp = campaign.NewRunner(&campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{"town": {Type: "town", NativeTownVariant: &v}}})
					g.campSel = s
					g.nativeTownUIPulse = c
					g.nativeShopMode = ""
				})
			}
		}
	}
	for _, art := range []bool{false, true} {
		for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD} {
			for variant := 0; variant < 3; variant++ {
				art, mode, variant := art, mode, variant
				p.captureCases[len(p.cases)] = true
				p.cases = append(p.cases, func() {
					g.modernStoryPortraits, g.interfaceMode = nil, mode
					if art {
						g.modernStoryPortraits = set
					}
					g.camp = campaign.NewRunner(&campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{"town": {Type: "town", NativeTownVariant: &variant}}})
					g.campSel, g.nativeTownUIPulse, g.nativeShopMode = 2, variant, ""
				})
			}
		}
	}
	for _, variant := range []int{1, 3, 5} {
		v := variant
		for _, mode := range []string{"menu", "purchase", "confirm", "insufficient"} {
			mode := mode
			p.cases = append(p.cases, func() {
				g.partyRoster[g.partyJoinOrder[0]] = originalFirst
				g.camp = campaign.NewRunner(&campaign.Campaign{Start: "shop", Nodes: map[string]*campaign.Node{"shop": {Type: "shop", NativeHubVariant: v, Goods: []campaign.Good{{ID: 0, Name: "item0", Price: 100}}}}})
				if !g.setupNativeShop() {
					p.err = fmt.Errorf("setup shop %d", v)
					return
				}
				g.nativeShopUIJob = nil
				g.nativeShopMode = mode
				g.shopSel = 0
			})
		}
		plan, err := campaign.PlanNativeShopPurchaseSuccess(v)
		if err != nil {
			t.Fatal(err)
		}
		timelineCount := plan.FrameCount + 1
		if plan.PreDelayBIOSTicks > 0 {
			timelineCount++
		}
		for step := 0; step < timelineCount; step++ {
			step := step
			p.cases = append(p.cases, func() {
				g.nativeShopMode = "success"
				timeline, ok := g.nativeShopSuccessTimeline()
				if !ok {
					p.err = fmt.Errorf("success %d", v)
					return
				}
				g.nativeShopUIJob = &nativeClassUIJob{timeline: timeline}
				for i := 0; i < step; i++ {
					g.nativeShopUIJob.elapsed += timeline[i].duration
				}
			})
		}
	}
	// Return to a real service2 equipment path with an existing HD party face.
	p.cases = append(p.cases, func() {
		g.camp = campaign.NewRunner(&campaign.Campaign{Start: "shop", Nodes: map[string]*campaign.Node{"shop": {Type: "shop", NativeHubVariant: 1, Goods: []campaign.Good{{ID: 0, Name: "item0", Price: 100}}}}})
		if !g.setupNativeShop() {
			p.err = fmt.Errorf("equip setup")
			return
		}
		unit := setNativeTransferInventory(g.partyRoster[0], 0, 1)
		g.partyRoster[0] = unit
		g.shopItemTypes[0], g.shopItemTypes[1] = 0, 0
		g.shopEquipTypes[1] = []int{0}
		g.shopItemStats = map[int]campaign.ItemStats{0: {Type: 0, AP: 1}, 1: {Type: 0, AP: 3}}
		screen := ebiten.NewImage(640, 400)
		defer screen.Dispose()
		drainNativeShopProductionJobs(t, g, screen)
		enterNativeShopEquipPanel(t, g, 0)
		drainNativeShopProductionJobs(t, g, screen)
		g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
		drainNativeShopProductionJobs(t, g, screen)
		settleNativeShopEquipPanelOpening(g)
		if g.nativeShopMode != "equip_panel" {
			p.err = fmt.Errorf("equip mode %s", g.nativeShopMode)
		}
	})
	for step := 0; step < 12; step++ {
		step := step
		p.cases = append(p.cases, func() { g.itemAnimStep = step; g.itemClosing = false })
	}
	gold, rng := g.gold, g.nativeRNGState
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != len(p.cases) {
		t.Fatalf("checks %d/%d", len(p.checks), len(p.cases))
	}
	if g.gold != gold || g.nativeRNGState != rng {
		t.Fatal("HD rendering changed gold/RNG")
	}
	if len(g.modernHDPanelPortraits) == 0 || g.modernHDPanelPortraits[0].portrait != set.portraits[0] {
		t.Fatal("existing Sol HD face absent from equipment panel")
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-FACILITY-GPU", "scope": "fixture Game.Draw; all 72 town selector/pulses, three shops, existing HD equipment portrait, entire toolbar viewport unchanged", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
