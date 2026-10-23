package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func healHDTestGame(t *testing.T, id int, multi bool) (*Game, *battle.Unit, *battle.Unit, []*battle.Unit) {
	t.Helper()
	g, actor, target := player9MapTestGame(t, true)
	actor.MP = 80
	actor.SetMapPlacement(10, 7, 3)
	target.SetMapPlacement(11, 7, 0)
	target.Camp = battle.Own
	target.NativeRecordByte6 = 0
	target.HasNativeRecordByte6 = true
	target.HP, target.MaxHP, target.NativeRecordWord42, target.HasNativeRecordWord42 = 99, 100, 100, true
	for _, u := range []*battle.Unit{actor, target} {
		if err := u.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
	}
	record := battle.NativeCommandRecord{ID: id, Damage: 70, SelectionMode: 1, EffectMode: 0, MPCost: 3, TargetCode: 1}
	if multi {
		extra := g.st.Units[5]
		extra.Camp, extra.OnField, extra.HP, extra.MaxHP = battle.Own, true, 20, 100
		extra.NativeRecordByte6, extra.HasNativeRecordByte6 = 0, true
		extra.NativeRecordWord42, extra.HasNativeRecordWord42 = 100, true
		extra.SetMapPlacement(11, 6, 0)
		if err := extra.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
		record.EffectMode = 1
	}
	g.st.NativeCommandBook[id] = record
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 11, CursorY: 7, VisibleCursorX: 11, VisibleCursorY: 7}); err != nil {
		t.Fatal(err)
	}
	g.curX, g.curY = 11, 7
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	targets, err := g.st.NativeCommandHealTargets(actor, target, id)
	if err != nil {
		t.Fatal(err)
	}
	want := 1
	if multi {
		want = 3 // The adjacent caster also lies in the one-cell effect area.
	}
	if len(targets) != want {
		t.Fatalf("heal fixture targets%d want%d", len(targets), want)
	}
	return g, actor, target, targets
}

func healHDTestTheme(t *testing.T) *modernStoryPortraitSet {
	t.Helper()
	catalog, pack := os.Getenv("FD2_TEST_HEAL_CATALOG"), os.Getenv("FD2_TEST_HEAL_PACK")
	if catalog == "" || pack == "" {
		t.Fatal("explicit full healing catalog/pack required")
	}
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func startHealHDTest(t *testing.T, g *Game, actor, target *battle.Unit, targets []*battle.Unit, id int, calls, done *int) {
	t.Helper()
	err := g.startNativeCommandHealPresentation(id, targets, func() ([]battle.NativeCommandHealResult, error) {
		*calls++
		r, next, err := g.st.ExecuteNativeCommandHealNative(actor, target, id, g.nativeRNGState)
		if err == nil {
			g.nativeRNGState = next
		}
		return r, err
	}, func([]battle.NativeCommandHealResult) { *done++ })
	if err != nil {
		t.Fatal(err)
	}
}

func TestModernHDHealNativeSourcesAndGeometry(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	frames, points := 0, 0
	for _, id := range []int{13, 14, 15, 16} {
		g, actor, target, targets := healHDTestGame(t, id, false)
		calls, done := 0, 0
		startHealHDTest(t, g, actor, target, targets, id, &calls, &done)
		j := g.nativeHealPresentation
		if j.hdInputErr != nil {
			t.Fatal(j.hdInputErr)
		}
		for i, pass := range j.hdPasses {
			work, vga := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, 64000)
			if err := indexedmap.ComposeNativeTransitionFrame(work, vga, nativeTransitionInput(j.hdInput), pass, j.hdLUTs[i]); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(vga, j.frontFrames[i]) {
				t.Fatal("frozen front producer differs", id, i)
			}
			frames++
			tc, oc, err := healHDCounts(pass)
			if err != nil {
				t.Fatal(err)
			}
			inside := func(x, y int, cx, cy, radius, scale, first, last, width int) bool {
				dy := math.Abs(float64(cy - y))
				if y < first || y >= last || dy >= float64(radius) {
					return false
				}
				span := int(math.Sqrt(float64(radius*radius)-dy*dy) * float64(scale) / 10)
				return x >= 0 && x < width && x >= cx-span && x < cx+span
			}
			for y := 0; y < 192; y += 3 {
				for x := 0; x < 312; x += 3 {
					at := 0x8088 + y*456 + x
					wx, wy := at%456, at/456
					a, b, c := pass.FirstRadial, pass.SecondRadial, pass.FinalRect
					first, second, rect := inside(wx, wy, a.CenterX, a.CenterY, a.Radius, a.Scale, a.StartY, a.EndY, a.ClipWidth), inside(wx, wy, b.CenterX, b.CenterY, b.Radius, b.Scale, b.StartY, b.EndY, b.ClipWidth), wy >= c.StartY && wy < c.EndY && wx >= 0 && wx < c.ClipWidth && wx >= c.CenterX-c.HorizontalRadius && wx < c.CenterX+c.HorizontalRadius
					wantT, wantO := byte(0), byte(0)
					if first {
						wantT++
					}
					if second {
						wantT++
						wantO++
					}
					if rect {
						wantT++
						wantO++
					}
					if healHDCountAt(tc, x, y) != wantT || healHDCountAt(oc, x, y) != wantO {
						t.Fatalf("geometry cmd%d frame%d %d,%d", id, i, x, y)
					}
					points++
				}
			}
		}
		proxy := &nativeCommand9PlayerJob{baselineWork: j.baselineWork, baselineVGA: j.baselineVGA}
		for i, sources := range j.effectSources {
			if len(sources) != 1 || sources[0].Entry != 57+i || !bytes.Equal(independentPlayer9VGA(t, g, proxy, sources), j.effectFrames[i]) {
				t.Fatal("effect producer differs", i)
			}
			frames++
		}
		for g.nativeHealPresentation != nil {
			if j.phase == nativeCommandHealDigitFrames {
				proxy.baselineWork, proxy.baselineVGA = j.baselineWork, j.baselineVGA
				for i, s := range j.digitSources {
					if !bytes.Equal(independentPlayer9VGA(t, g, proxy, s), j.digitFrames[i]) {
						t.Fatal("digit producer differs", i)
					}
					frames++
				}
				break
			}
			j.drawn = true
			g.stepNativeCommandHealPresentation()
		}
		if calls != 1 || done != 0 || target.HP != 100 || actor.MP != 77 || len(j.digitSources) != 22 {
			t.Fatal("transaction or capped heal changed")
		}
	}
	t.Logf("%d independent raw-work geometry points; %d complete front/effect/digit canvases, four schedules and one capped transaction", points, frames)
}

func TestModernHDHealLMIAtomic(t *testing.T) {
	if os.Getenv("FD2_HD_HEAL_UNIT") != "1" {
		t.Skip("explicit healing catalog/pack")
	}
	pack := os.Getenv("FD2_TEST_HEAL_PACK")
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_HEAL_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	groups := 0
	for _, asset := range catalog.Assets {
		if asset.Role != "map_overlay_hd_set" || !(asset.SourceResource == 5 && asset.FrameCount == 10 || asset.SourceResource == 6 && asset.FrameCount == 7) {
			continue
		}
		groups++
		if err := loadModernMapOverlayHDAsset(&modernStoryPortraitSet{}, asset, pack); err != nil {
			t.Fatal(err)
		}
		for _, fault := range []string{"metadata", "source", "contract", "count", "descriptor", "missing", "sha", "path", "png-size"} {
			bad := asset
			bad.Files = append([]string(nil), asset.Files...)
			bad.FrameSHA256 = append([]string(nil), asset.FrameSHA256...)
			bad.NativeLMIFrames = append([]modernLMIHDDescriptor(nil), asset.NativeLMIFrames...)
			root := pack
			switch fault {
			case "metadata":
				bad.SourceMetadataSHA256 = "bad"
			case "source":
				bad.SourceResource = 4
			case "contract":
				bad.ConsumerContract = "unknown"
			case "count":
				bad.FrameCount++
			case "descriptor":
				bad.NativeLMIFrames[0].PixelsSHA256 = "bad"
			case "missing":
				bad.Files[len(bad.Files)-1] = "absent-heal.png"
			case "sha":
				bad.FrameSHA256[0] = "bad"
			case "path":
				bad.Files[0] = "../escape.png"
			case "png-size":
				root = t.TempDir()
				for _, n := range asset.Files {
					if err := os.Symlink(filepath.Join(pack, n), filepath.Join(root, n)); err != nil {
						t.Fatal(err)
					}
				}
				n := asset.Files[len(asset.Files)-1]
				if err := os.Remove(filepath.Join(root, n)); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(filepath.Join(root, n))
				if err != nil {
					t.Fatal(err)
				}
				if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 6, 6))); err != nil {
					t.Fatal(err)
				}
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
				b, err := os.ReadFile(filepath.Join(root, n))
				if err != nil {
					t.Fatal(err)
				}
				bad.FrameSHA256[len(bad.FrameSHA256)-1] = fmt.Sprintf("%x", sha256.Sum256(b))
			}
			marker := &modernMapOverlayHDResource{frames: map[int]*image.RGBA{999: image.NewRGBA(image.Rect(0, 0, 1, 1))}}
			set := &modernStoryPortraitSet{mapEffects: marker, mapOverlays: marker}
			if err := loadModernMapOverlayHDAsset(set, bad, root); err == nil || set.mapEffects != marker || set.mapOverlays != marker {
				t.Fatal("invalid healing group published", asset.SourceResource, fault)
			}
		}
	}
	if groups != 2 {
		t.Fatal("missing two healing source groups")
	}
	t.Log("all17 real LMI PNGs plus18 invalid cold-group cases passed atomically")
}

func TestModernHDHealFullThemeAndAtomic(t *testing.T) {
	if os.Getenv("FD2_HD_HEAL_UNIT") != "1" {
		t.Skip("dedicated complete HD theme process")
	}
	t.Setenv("FD2_MUTE", "1")
	set := healHDTestTheme(t)
	for _, id := range []int{13, 14, 15, 16} {
		g, actor, target, targets := healHDTestGame(t, id, false)
		g.modernStoryPortraits = set
		calls, done := 0, 0
		startHealHDTest(t, g, actor, target, targets, id, &calls, &done)
		j := g.nativeHealPresentation
		if j.hdFrames == nil || len(j.hdFrames.Front) != 16 || len(j.hdFrames.Effects) != 7 || calls != 0 || actor.MP != 80 || target.HP != 99 {
			t.Fatal("incomplete HD owner or early transaction")
		}
		normal, err := g.prepareModernHDCommandSteady(j.hdInput, j.baselineWork, j.baselineVGA, j.palette, set)
		if err != nil {
			t.Fatal(err)
		}
		inside, changed, outsideMismatch := 0, 0, 0
		for index, front := range j.hdFrames.Front {
			counts, _, err := healHDCounts(j.hdPasses[index])
			if err != nil {
				t.Fatal(err)
			}
			for k, tile := range front.Terrain {
				base := normal.Terrain[k]
				for y := 0; y < 144; y++ {
					for x := 0; x < 144; x++ {
						xx, yy := tile.Rect.Min.X+x, tile.Rect.Min.Y+y
						if xx < 0 || xx >= 1872 || yy < 0 || yy >= 1152 {
							continue
						}
						if healHDCountAt(counts, xx/6, yy/6) > 0 {
							inside++
							if tile.Image.RGBAAt(x, y) != base.Image.RGBAAt(x, y) {
								changed++
							}
						} else if tile.Image.RGBAAt(x, y) != base.Image.RGBAAt(x, y) {
							outsideMismatch++
						}
					}
				}
			}
		}
		if inside < 100 || changed < 100 || outsideMismatch != 0 {
			t.Fatalf("heal%d HD region inside%d changed%d outside-mismatch%d", id, inside, changed, outsideMismatch)
		}
		t.Logf("heal%d HD region inside=%d changed=%d outside_mismatch=%d", id, inside, changed, outsideMismatch)
		if err := g.applyTheme(nil); err != nil {
			t.Fatal(err)
		}
		for _, entry := range []int{57, 63, 105, 114} {
			bad := clonePlayer9MapTheme(set)
			bank := bad.mapEffects
			if entry >= 105 {
				bank = bad.mapOverlays
			}
			delete(bank.frames, entry)
			if err := g.applyTheme(bad); err == nil || g.modernStoryPortraits != nil || j.phase != nativeCommandHealFrames || j.frame != 0 || calls != 0 || actor.MP != 80 || target.HP != 99 {
				t.Fatal("bad F2 published or mutated state")
			}
		}
		bad := clonePlayer9MapTheme(set)
		bad.mapEffects.frames[57] = image.NewRGBA(image.Rect(0, 0, 6, 6))
		if err := g.applyTheme(bad); err == nil {
			t.Fatal("wrong source dimensions accepted")
		}
		if err := g.applyTheme(set); err != nil {
			t.Fatal(err)
		}
		if j.hdFrames == nil || j.hdTheme != set || calls != 0 {
			t.Fatal("good F2 unavailable")
		}
		// No complete group declared: preserve HD world with native LMI fallback.
		legacy := clonePlayer9MapTheme(set)
		for i := 57; i <= 63; i++ {
			delete(legacy.mapEffects.frames, i)
			delete(legacy.mapEffects.native, i)
		}
		for i := 105; i <= 114; i++ {
			delete(legacy.mapOverlays.frames, i)
			delete(legacy.mapOverlays.native, i)
		}
		if _, err := g.prepareModernHDHeal(j, legacy); err != nil {
			t.Fatal("old theme fallback rejected", err)
		}
	}
}
