package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func townHDDepartureTestOwner(t *testing.T, g *Game, variant int) *nativeTownDepartureJob {
	t.Helper()
	g.nativeTownDeparture = nil
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{
		"town": {Type: "town", NativeTownVariant: &variant},
		"prep": {Type: "preparation", Cancel: "town"},
	}})
	g.campSel, g.nativeTownUIPulse = 2, 3
	unit := g.partyRoster[g.partyJoinOrder[0]]
	unit.MapSelectorKey, unit.HasMapSelectorKey = 50, true
	g.partyRoster[g.partyJoinOrder[0]] = unit
	g.captureNativeTownCaller()
	if g.prepTownSource == nil || len(g.prepPromptSource) != 64000 {
		t.Fatal("town caller capture unavailable")
	}
	g.camp.Cur = "prep"
	g.nativeMapAssets, _, _ = completeNativeMapFrameFixture(t)
	g.nativeMapAssets.PaletteDAC = append([]byte(nil), g.nativeClassUI.paletteDAC...)
	g.nativeMapAssets.Palette = g.nativeClassUI.palette
	g.nativeMapDAC = append([]byte(nil), g.nativeClassUI.paletteDAC...)
	if err := g.startNativeTownDeparture(nil); err != nil {
		t.Fatal(err)
	}
	return g.nativeTownDeparture
}

func townHDDepartureCacheSHA(job *nativeTownDepartureJob) [32]byte {
	h := sha256.New()
	for _, frame := range job.frames {
		h.Write(frame.indexed)
		h.Write(frame.dac)
		for _, c := range frame.palette {
			r, g, b, a := c.RGBA()
			h.Write([]byte{byte(r >> 8), byte(g >> 8), byte(b >> 8), byte(a >> 8)})
		}
	}
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

func TestModernHDTownDepartureFrozenSourceAndStyles(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	for variant := 0; variant < 3; variant++ {
		job := townHDDepartureTestOwner(t, g, variant)
		d := job.source
		if d == nil || d.group != 50 || d.pulse != 3 {
			t.Fatal("missing resolved caller")
		}
		before := townHDDepartureCacheSHA(job)
		source := append([]byte(nil), g.prepPromptSource...)
		// Live party, selection and town node cease to describe this caller.
		unit := g.partyRoster[g.partyJoinOrder[0]]
		unit.MapSelectorKey = 118
		g.partyRoster[g.partyJoinOrder[0]] = unit
		g.campSel, g.nativeTownUIPulse = 0, 2
		*g.camp.C.Nodes["town"].NativeTownVariant = (variant + 1) % 3
		for step := 0; step < 11; step++ {
			job.step = step
			for _, art := range []bool{false, true} {
				for _, ui := range []string{playerInterfaceOriginal, playerInterfaceHD} {
					g.modernStoryPortraits = nil
					if art {
						g.modernStoryPortraits = set
					}
					g.interfaceMode = ui
					g.Draw(screen)
					if !job.drawn || job.step != step {
						t.Fatal("Draw advanced departure")
					}
					if (g.modernHDScene != nil) != (art && step < 10) {
						t.Fatalf("wrong backdrop: variant=%d step=%d art=%v ui=%s", variant, step, art, ui)
					}
					if o := g.modernHDScene; o != nil && (o.cursor == nil || o.cursor.group != 50 || o.cursor.frame != 1) {
						t.Fatal("departure retargeted the live party")
					}
					if ui == playerInterfaceHD && step == 0 && (g.modernHDText == nil || len(g.modernHDText.runs) == 0) {
						t.Fatal("independent HD interface disappeared")
					}
					if step == 10 && g.modernHDText != nil && len(g.modernHDText.runs) != 0 {
						t.Fatal("black boundary retained text")
					}
				}
			}
		}
		if before != townHDDepartureCacheSHA(job) || !bytes.Equal(source, g.prepPromptSource) {
			t.Fatal("display changed native indexed/DAC cache or caller")
		}
	}
	g.modernStoryPortraits = set
	job := townHDDepartureTestOwner(t, g, 2)
	job.step = 0
	before := townHDDepartureCacheSHA(job)
	for i := 0; i < 2; i++ {
		previousArt, previousUI := g.modernStoryPortraits, g.interfaceMode
		g.cycleTheme()
		g.cycleInterfaceMode()
		g.Draw(screen)
		if g.loadErr != "" || g.nativeTownDeparture != job || job.step != 0 || before != townHDDepartureCacheSHA(job) {
			t.Fatal("F2/F11 changed departure owner/cache", g.loadErr)
		}
		if (previousArt == nil) == (g.modernStoryPortraits == nil) || previousUI == g.interfaceMode {
			t.Fatal("F2/F11 did not switch", g.msg)
		}
	}
	for _, fault := range []string{"source-mismatch", "missing-descriptor", "missing-hd", "bad-geometry"} {
		t.Run(fault, func(t *testing.T) {
			g.nativeTownDeparture = nil
			originalSource, descriptor, pictures := g.prepPromptSource, g.prepTownSource, set.mapSpritesHD[50]
			defer func() {
				g.prepPromptSource, g.prepTownSource, set.mapSpritesHD[50] = originalSource, descriptor, pictures
			}()
			switch fault {
			case "source-mismatch":
				g.prepPromptSource = append([]byte(nil), originalSource...)
				g.prepPromptSource[0] ^= 255
			case "missing-descriptor":
				g.prepTownSource = nil
			case "missing-hd":
				delete(set.mapSpritesHD, 50)
			case "bad-geometry":
				set.mapSpritesHD[50] = []image.Image{nil, image.NewNRGBA(image.Rect(0, 0, 72, 72))}
			}
			if err := g.startNativeTownDeparture(nil); err != nil {
				t.Fatal(err)
			}
			if g.drawModernHDTownDeparture(screen, g.nativeTownDeparture) {
				t.Fatal("accepted", fault)
			}
			g.Draw(screen)
			if !g.nativeTownDeparture.drawn {
				t.Fatal("fallback lost native acknowledgement")
			}
		})
	}
}

type townHDDepartureProbe struct {
	t            *testing.T
	g            *Game
	set          *modernStoryPortraitSet
	presentation *modernHDPresentation
	index        int
	drawn        bool
	err          error
	out          string
	checks       []map[string]any
	jobs         [3]*nativeTownDepartureJob
}

func (p *townHDDepartureProbe) Layout(int, int) (int, int) { return 640, 424 }
func (p *townHDDepartureProbe) Update() error {
	if p.err != nil {
		return ebiten.Termination
	}
	if p.drawn {
		p.index++
		p.drawn = false
	}
	if p.index == 132 {
		return ebiten.Termination
	}
	variant, step, style := p.index/44, p.index%44/4, p.index%4
	p.g.nativeTownDeparture = p.jobs[variant]
	p.g.nativeTownDeparture.step = step
	p.g.modernStoryPortraits = nil
	if style >= 2 {
		p.g.modernStoryPortraits = p.set
	}
	p.g.interfaceMode = playerInterfaceOriginal
	if style%2 == 1 {
		p.g.interfaceMode = playerInterfaceHD
	}
	return nil
}
func (p *townHDDepartureProbe) Draw(screen *ebiten.Image) {
	before, _ := json.Marshal(p.g.partyRoster)
	gold, rng, step := p.g.gold, p.g.nativeRNGState, p.g.nativeTownDeparture.step
	p.presentation.Draw(screen)
	after, _ := json.Marshal(p.g.partyRoster)
	if !bytes.Equal(before, after) || gold != p.g.gold || rng != p.g.nativeRNGState || step != p.g.nativeTownDeparture.step {
		p.err = fmt.Errorf("departure Draw changed party/gold/RNG/owner step")
	}
}

// Independent bilinear sampling uses HD source pixel centers. The production
// path uses GPU geometry and does not call this reference compositor.
func townHDBilinear(picture image.Image, x, y float64) [3]float64 {
	var out [3]float64
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			weight := (1 - math.Abs(x-float64(x0+dx))) * (1 - math.Abs(y-float64(y0+dy)))
			c := color.NRGBAModel.Convert(picture.At(x0+dx, y0+dy)).(color.NRGBA)
			out[0] += float64(c.R) * weight
			out[1] += float64(c.G) * weight
			out[2] += float64(c.B) * weight
		}
	}
	return out
}

func townHDCore(picture image.Image, x, y float64) bool {
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			if !image.Pt(x0+dx, y0+dy).In(picture.Bounds()) {
				return false
			}
			_, _, _, a := picture.At(x0+dx, y0+dy).RGBA()
			if a < 250*257 {
				return false
			}
		}
	}
	return true
}

func townHDRGBACore(picture image.Image, x, y float64) [4]float64 {
	var result [4]float64
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			weight := (1 - math.Abs(x-float64(x0+dx))) * (1 - math.Abs(y-float64(y0+dy)))
			c := color.NRGBAModel.Convert(picture.At(x0+dx, y0+dy)).(color.NRGBA)
			a := float64(c.A) / 255
			result[0] += float64(c.R) * a * weight
			result[1] += float64(c.G) * a * weight
			result[2] += float64(c.B) * a * weight
			result[3] += a * weight
		}
	}
	if result[3] > 0 {
		for i := 0; i < 3; i++ {
			result[i] /= result[3]
		}
	}
	return result
}

// Ebiten v2.6.6 internal/graphics/vertex.go snaps destination vertices to
// 0, 5/16, 11/16 or the next pixel. Account for that pinned renderer contract
// before independent source sampling; a smooth affine reference is different.
func townHDSnap(value float64) float64 {
	x := float32(value)
	i := float32(math.Floor(float64(x)))
	f := x - i
	switch {
	case f < 3.0/16.0:
		return float64(i)
	case f < 8.0/16.0:
		return float64(i + 5.0/16.0)
	case f < 13.0/16.0:
		return float64(i + 11.0/16.0)
	default:
		return float64(i + 1)
	}
}

func townHDSourceCoordinate(pixel int, start, end float64, size int) float64 {
	left, right := townHDSnap(start), townHDSnap(end)
	return (float64(pixel)+0.5-left)/(right-left)*float64(size) - 0.5
}

func (p *townHDDepartureProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.presentation.DrawFinalScreen(screen, offscreen, geometry)
	if p.drawn || p.err != nil {
		return
	}
	final := ebiten.NewImage(1920, 1272)
	defer final.Dispose()
	var scale ebiten.GeoM
	scale.Scale(3, 3)
	p.presentation.DrawFinalScreen(final, offscreen, scale)
	pixels := make([]byte, 1920*1272*4)
	final.ReadPixels(pixels)
	native := make([]byte, 640*424*4)
	offscreen.ReadPixels(native)
	variant, step, style := p.index/44, p.index%44/4, p.index%4
	checked, nativeChecks, cursorChecks := 0, 0, 0
	if style >= 2 && step < 10 {
		o := p.g.modernHDScene
		if o == nil || o.departure == nil || o.cursor == nil || o.cursor.group != 50 {
			p.err = fmt.Errorf("missing frozen HD scene %d", p.index)
			return
		}
		geom, err := campaign.NativeTownDepartureGeometryFor(variant, 2, step+1)
		if err != nil {
			p.err = err
			return
		}
		magnification := 128 / float64(geom.Spacing)
		left := 960 + (24*128-6*float64(geom.Center.X))/float64(geom.Spacing)
		top := 600 + (24*128-6*float64(geom.Center.Y))/float64(geom.Spacing)
		for y := 6; y < 1194; y += 17 {
			for x := 6; x < 1914; x += 17 {
				lx, ly := x/3, y/3+24
				clear := true
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if native[((ly+dy)*640+lx+dx)*4+3] != 0 {
							clear = false
						}
					}
				}
				if !clear {
					continue
				}
				sx := townHDSourceCoordinate(x, left, left+1872*magnification, 1872)
				sy := townHDSourceCoordinate(y, top, top+1152*magnification, 1152)
				if image.Pt(int(sx+24), int(sy+24)).In(o.cursor.rect.Inset(-4)) {
					continue
				}
				if sx < 2 || sy < 2 || sx > 1870 || sy > 1150 {
					continue
				}
				want := townHDBilinear(o.scene.image, sx, sy)
				at := ((y+72)*1920 + x) * 4
				for c := 0; c < 3; c++ {
					v := math.Max(0, want[c]-255*float64(4*(step+1))/63)
					if math.Abs(float64(pixels[at+c])-v) > 2 {
						f, _ := os.Create(filepath.Join(p.out, "failed-gpu.png"))
						if f != nil {
							_ = png.Encode(f, &image.RGBA{Pix: pixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1272)})
							_ = f.Close()
						}
						p.err = fmt.Errorf("HD sample case%d (%d,%d) channel%d got%d want%.2f", p.index, x, y, c, pixels[at+c], v)
						return
					}
				}
				checked++
			}
		}
		if checked < 1000 {
			p.err = fmt.Errorf("too few HD samples %d: %d", p.index, checked)
			return
		}
		cursor := p.set.mapSpritesHD[o.cursor.group][o.cursor.frame]
		visible := o.cursor.rect.Intersect(o.cursor.clip)
		cursorLeft := 960 + (float64(visible.Min.X)*128-6*float64(geom.Center.X))/float64(geom.Spacing)
		cursorTop := 600 + (float64(visible.Min.Y)*128-6*float64(geom.Center.Y))/float64(geom.Spacing)
		for y := max(0, int(math.Ceil(cursorTop))); y < min(1200, int(math.Floor(cursorTop+float64(visible.Dy())*magnification))); y += 3 {
			for x := max(0, int(math.Ceil(cursorLeft))); x < min(1920, int(math.Floor(cursorLeft+float64(visible.Dx())*magnification))); x += 3 {
				if native[((y/3+24)*640+x/3)*4+3] != 0 {
					continue
				}
				cx := townHDSourceCoordinate(x, cursorLeft, cursorLeft+float64(visible.Dx())*magnification, visible.Dx()) + float64(visible.Min.X-o.cursor.rect.Min.X)
				cy := townHDSourceCoordinate(y, cursorTop, cursorTop+float64(visible.Dy())*magnification, visible.Dy()) + float64(visible.Min.Y-o.cursor.rect.Min.Y)
				if !townHDCore(cursor, cx, cy) {
					continue
				}
				want := townHDRGBACore(cursor, cx, cy)
				sx := townHDSourceCoordinate(x, left, left+1872*magnification, 1872)
				sy := townHDSourceCoordinate(y, top, top+1152*magnification, 1152)
				background := townHDBilinear(o.scene.image, sx, sy)
				at := ((y+72)*1920 + x) * 4
				for c := 0; c < 3; c++ {
					amount := 255 * float64(4*(step+1)) / 63
					v := math.Max(0, want[c]-amount)*want[3] + math.Max(0, background[c]-amount)*(1-want[3])
					if math.Abs(float64(pixels[at+c])-v) > 2 {
						p.err = fmt.Errorf("cursor sample case%d (%d,%d) got%d want%.2f", p.index, x, y, pixels[at+c], v)
						return
					}
				}
				cursorChecks++
			}
		}
		if cursorChecks < 200 {
			p.err = fmt.Errorf("too few cursor cores case%d: %d", p.index, cursorChecks)
			return
		}
	}
	if style%2 == 0 || step == 10 {
		for y := 0; y < 400; y++ {
			for x := 0; x < 640; x++ {
				at := ((y+24)*640 + x) * 4
				if native[at+3] != 255 {
					continue
				}
				atFinal := (((y+24)*3+1)*1920 + x*3 + 1) * 4
				if !bytes.Equal(native[at:at+4], pixels[atFinal:atFinal+4]) {
					p.err = fmt.Errorf("native UI sample changed case%d (%d,%d)", p.index, x, y)
					return
				}
				nativeChecks++
			}
		}
	}
	if step == 10 {
		for i := 72 * 1920 * 4; i < len(pixels); i += 4 {
			if pixels[i] != 0 || pixels[i+1] != 0 || pixels[i+2] != 0 {
				p.err = fmt.Errorf("black boundary leaked case%d", p.index)
				return
			}
		}
	}
	if step == 0 && style%2 == 1 && (p.g.modernHDText == nil || len(p.g.modernHDText.runs) == 0) {
		p.err = fmt.Errorf("independent HD text missing")
		return
	}
	if (step == 0 || step == 9 || step == 10) && (style == 0 || style == 3) {
		path := filepath.Join(p.out, fmt.Sprintf("town-v%d-step%02d-style%d.png", variant, step+1, style))
		f, err := os.Create(path)
		if err == nil {
			err = png.Encode(f, &image.RGBA{Pix: pixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1272)})
			if e := f.Close(); err == nil {
				err = e
			}
		}
		if err != nil {
			p.err = err
			return
		}
	}
	p.checks = append(p.checks, map[string]any{"case": p.index, "variant": variant, "step": step + 1, "style": style, "hd_samples": checked, "native_ui_samples": nativeChecks, "cursor_core_samples": cursorChecks})
	p.drawn = true
}

func TestModernHDTownDepartureRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_TOWN_DEPARTURE_GPU") != "1" {
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
	p := &townHDDepartureProbe{t: t, g: g, set: set, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true}}
	var before [3][32]byte
	for v := 0; v < 3; v++ {
		p.jobs[v] = townHDDepartureTestOwner(t, g, v)
		before[v] = townHDDepartureCacheSHA(p.jobs[v])
	}
	unit := g.partyRoster[g.partyJoinOrder[0]]
	unit.MapSelectorKey = 118
	g.partyRoster[g.partyJoinOrder[0]] = unit
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != 132 {
		t.Fatal("incomplete GPU matrix", len(p.checks))
	}
	for v := 0; v < 3; v++ {
		if townHDDepartureCacheSHA(p.jobs[v]) != before[v] {
			t.Fatal("native cache changed", v)
		}
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-TOWN-DEPARTURE-GPU", "scope": "fixture complete Game.Draw; 3 variants, 11 boundaries, independent 4 styles; no new original E2", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
