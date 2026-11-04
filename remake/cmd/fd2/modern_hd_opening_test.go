package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

var openingTestSet *modernStoryPortraitSet
var openingTestAssets []modernThemeAsset

func openingHDTestAssets(t *testing.T) (*modernStoryPortraitSet, []modernThemeAsset, string) {
	t.Helper()
	path, root := os.Getenv("FD2_TEST_OPENING_CATALOG"), os.Getenv("FD2_TEST_OPENING_PACK")
	if path == "" || root == "" {
		t.Skip("dedicated opening verifier supplies catalog and immutable pack")
	}
	if openingTestSet != nil {
		return openingTestSet, openingTestAssets, root
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	set := &modernStoryPortraitSet{}
	var assets []modernThemeAsset
	for _, asset := range catalog.Assets {
		if asset.Role == modernOpeningRole {
			if err := loadModernOpeningHDAsset(set, asset, root); err != nil {
				t.Fatal(err)
			}
			assets = append(assets, asset)
		} else if asset.Role == "story_animation_hd_set" && asset.SourceArchive == "ANI.DAT" && asset.SourceResource == 0 {
			if err := loadModernSkyKeyHDAsset(set, asset, root); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := validateModernOpeningHDGroup(set); err != nil || set.openingHD == nil {
		t.Fatalf("complete opening required: %v", err)
	}
	openingTestSet, openingTestAssets = set, assets
	return set, assets, root
}

func TestModernHDOpeningAdmission(t *testing.T) {
	set, assets, root := openingHDTestAssets(t)
	if len(assets) != 12 {
		t.Fatal("complete original scene collection required")
	}
	var one modernThemeAsset
	for _, asset := range assets {
		if asset.SourceArchive == "FDOTHER.DAT" && asset.SourceResource == 100 {
			one = asset
		}
	}
	for _, kind := range []string{"geometry", "source", "hash", "path", "duplicate", "partial", "transparent", "corrupt"} {
		t.Run(kind, func(t *testing.T) {
			raw, _ := json.Marshal(one)
			var asset modernThemeAsset
			if err := json.Unmarshal(raw, &asset); err != nil {
				t.Fatal(err)
			}
			target, directory := &modernStoryPortraitSet{}, root
			switch kind {
			case "geometry":
				asset.Width--
			case "source":
				asset.SourceMetadataSHA256 = "wrong"
			case "hash":
				asset.FrameSHA256[0] = "wrong"
			case "path":
				asset.Files[0] = "../outside.png"
			case "duplicate":
				target.openingHD = &modernOpeningHD{banks: map[modernOpeningKey]*modernOpeningBank{{asset.SourceArchive, asset.SourceResource}: {}}}
			case "partial":
				target.openingHD = &modernOpeningHD{banks: map[modernOpeningKey]*modernOpeningBank{{asset.SourceArchive, asset.SourceResource}: {}}}
				if validateModernOpeningHDGroup(target) == nil {
					t.Fatal("partial opening admitted")
				}
				return
			case "transparent", "corrupt":
				directory = t.TempDir()
				for _, name := range asset.SourceFiles {
					data, err := os.ReadFile(filepath.Join(root, name))
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
						t.Fatal(err)
					}
				}
				data := []byte("broken PNG")
				if kind == "transparent" {
					picture := image.NewNRGBA(image.Rect(0, 0, 1920, 1200))
					draw.Draw(picture, picture.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
					picture.SetNRGBA(0, 0, color.NRGBA{})
					var output bytes.Buffer
					if err := png.Encode(&output, picture); err != nil {
						t.Fatal(err)
					}
					data = output.Bytes()
				}
				asset.FrameSHA256[0] = skyKeySHA(data)
				if err := os.WriteFile(filepath.Join(directory, asset.Files[0]), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if loadModernOpeningHDAsset(target, asset, directory) == nil {
				t.Fatal("invalid opening asset admitted")
			}
		})
	}
	if set.skyKeyANI == nil {
		t.Fatal("existing ANI0 must remain shared")
	}
}

type openingGPUCase struct {
	step, frame int
	archive     string
	resource    int
}

type openingGPUProbe struct {
	t     *testing.T
	g     *Game
	cases []openingGPUCase
	index int
	err   error
	final *ebiten.Image
	out   string
	rows  []map[string]any
}

func (p *openingGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *openingGPUProbe) Update() error {
	if p.err != nil || p.index >= len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *openingGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= len(p.cases) {
		return
	}
	c := p.cases[p.index]
	g := p.g
	g.cutIdx, g.cutFrame, g.cutTick, g.scrollY = c.step, c.frame, 0, 0
	if c.archive == "FDOTHER.DAT" && c.resource == 69 && c.frame == 1 {
		g.scrollY = 520
	}
	if cutScript[c.step].kind == "publisher" {
		g.cutTick = 64
	} else if cutScript[c.step].kind == "titlefade" {
		g.cutTick = 19
	}
	before := fmt.Sprint(g.cutIdx, g.cutFrame, g.cutTick, g.scrollY, g.titleSel, g.nativeRNGState)
	screen.Fill(color.RGBA{R: 255, A: 255})
	g.drawTitle(screen)
	logical := make([]byte, 640*400*4)
	screen.ReadPixels(logical)
	for _, value := range logical {
		if value != 0 {
			p.err = fmt.Errorf("case%d retained original title pixels over HD", p.index)
			return
		}
	}
	p.final.Clear()
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	presentation := &modernHDPresentation{Game: g}
	presentation.DrawFinalScreen(p.final, screen, geometry)
	actual := make([]byte, 1920*1200*4)
	p.final.ReadPixels(actual)
	var expected image.Image
	if c.archive == "ANI.DAT" && c.resource == 0 {
		expected, p.err = g.modernStoryPortraits.skyKeyANI.picture(c.frame)
	} else {
		bank := g.modernStoryPortraits.openingHD.banks[modernOpeningKey{c.archive, c.resource}]
		expected, p.err = png.Decode(bytes.NewReader(bank.encoded[c.frame]))
	}
	if p.err != nil {
		return
	}
	want := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	sourcePoint := image.Point{}
	if c.resource == 69 && c.archive == "FDOTHER.DAT" {
		sourcePoint.Y = int(g.scrollY * 6)
	}
	draw.Draw(want, want.Bounds(), expected, sourcePoint, draw.Src)
	for i, value := range actual {
		delta := int(value) - int(want.Pix[i])
		if delta < -1 || delta > 1 {
			p.err = fmt.Errorf("case%d final opening layer differs at byte%d: %d/%d", p.index, i, value, want.Pix[i])
			return
		}
	}
	if after := fmt.Sprint(g.cutIdx, g.cutFrame, g.cutTick, g.scrollY, g.titleSel, g.nativeRNGState); after != before {
		p.err = fmt.Errorf("drawing changed original title state: %s/%s", before, after)
		return
	}
	count := 96
	if !(c.archive == "ANI.DAT" && c.resource == 0) {
		count = len(g.modernStoryPortraits.openingHD.banks[modernOpeningKey{c.archive, c.resource}].encoded)
	}
	if p.out != "" && (c.frame == 0 || c.frame == count-1) {
		name := fmt.Sprintf("opening-step%02d-frame%03d.png", c.step, c.frame)
		file, err := os.Create(filepath.Join(p.out, name))
		if err == nil {
			err = png.Encode(file, &image.RGBA{Pix: actual, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1200)})
			file.Close()
		}
		if err != nil {
			p.err = err
			return
		}
	}
	p.rows = append(p.rows, map[string]any{"step": c.step, "archive": c.archive, "resource": c.resource, "frame": c.frame, "rgba_sha256": skyKeySHA(actual)})
	p.index++
}

func TestModernHDOpeningRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_TEST_OPENING_GPU") != "1" {
		t.Skip("dedicated GPU process required")
	}
	set, _, _ := openingHDTestAssets(t)
	ta, err := loadTitleAssets()
	if err != nil {
		t.Fatal(err)
	}
	p := &openingGPUProbe{t: t, g: &Game{modernStoryPortraits: set, titleAssets: ta, titlePhase: "cutscene"},
		final: ebiten.NewImage(1920, 1200), out: os.Getenv("FD2_TEST_OPENING_OUT")}
	if p.out != "" {
		if err := os.MkdirAll(p.out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for step, row := range cutScript {
		if only := os.Getenv("FD2_TEST_OPENING_ONLY_ANI"); only != "" && (row.kind != "afm" || fmt.Sprint(row.res) != only) {
			continue
		}
		if row.kind == "afm" {
			count := 96
			if row.res != 0 {
				_, _, count = modernOpeningShape(modernOpeningKey{"ANI.DAT", row.res})
			}
			for frame := 0; frame < count; frame++ {
				p.cases = append(p.cases, openingGPUCase{step, frame, "ANI.DAT", row.res})
			}
		} else if row.kind == "static" {
			p.cases = append(p.cases, openingGPUCase{step, 0, "FDOTHER.DAT", []int{100, 75}[row.res]})
		} else if row.kind == "publisher" {
			p.cases = append(p.cases, openingGPUCase{step, 0, "FDOTHER.DAT", 74})
		} else if row.kind == "scroll" || row.kind == "hold" || row.kind == "redfade" {
			p.cases = append(p.cases, openingGPUCase{step, 0, "FDOTHER.DAT", 69})
			if row.kind == "scroll" && row.scrollFrom == 535 {
				p.cases = append(p.cases, openingGPUCase{step, 1, "FDOTHER.DAT", 69})
			}
		} else if row.kind == "titlefade" {
			p.cases = append(p.cases, openingGPUCase{step, 0, "FDOTHER.DAT", 7})
		}
	}
	ebiten.SetWindowSize(1280, 800)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if p.index != len(p.cases) {
		t.Fatal("opening GPU cases incomplete")
	}
	if p.out != "" {
		raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-OPENING-GPU-SCOPED", "normal_player_verified": false, "state_injection": true,
			"scope": "直接控制原cutScript影格的GPU全幀抽測，不代表普通開場或DOS原版對拍。", "rows": p.rows}, "", "  ")
		if err := os.WriteFile(filepath.Join(p.out, "opening-gpu.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
