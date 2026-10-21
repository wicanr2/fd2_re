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
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
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
	if p.g.modernStoryPortraits != nil {
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
		p.checks = append(p.checks, map[string]any{"case": p.index, "hd_pixels": checked, "scene": p.g.camp.Node().Type, "variant": p.g.nativeShopVariant})
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
	scene := p.g.modernHDScene.scene
	if !p.captured[scene] || len(p.g.modernHDPanelPortraits) > 0 && !p.headCaptured || p.index == len(p.cases)-1 {
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
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}, captured: make(map[*modernFacilityScene]bool)}
	for variant := 0; variant < 3; variant++ {
		for selection := 0; selection < 6; selection++ {
			for pulse := 0; pulse < 4; pulse++ {
				v, s, c := variant, selection, pulse
				p.cases = append(p.cases, func() {
					g.camp = campaign.NewRunner(&campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{"town": {Type: "town", NativeTownVariant: &v}}})
					g.campSel = s
					g.nativeTownUIPulse = c
					g.nativeShopMode = ""
				})
			}
		}
	}
	for _, variant := range []int{1, 3, 5} {
		v := variant
		for _, mode := range []string{"menu", "purchase", "confirm", "insufficient"} {
			mode := mode
			p.cases = append(p.cases, func() {
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
