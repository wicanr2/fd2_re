package main

import (
	"encoding/json"
	"errors"
	"image"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// storyMaskBindingLayouts 從正式 cutscene binding 取每種 speaker control 列數最多的
// 原生對白，涵蓋多頁、三行窗口捲動與 FFEE 超出格網的文字窗。
func storyMaskBindingLayouts(t *testing.T) map[string]*campaign.NativeDialogueLayout {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(assetPath("assets/cutscenes/bindings"), "*.json"))
	if err != nil || len(files) == 0 {
		t.Skip("cutscene bindings unavailable")
	}
	best := map[string]*campaign.NativeDialogueLayout{}
	rows := func(l *campaign.NativeDialogueLayout) int {
		n := 0
		for _, page := range l.Pages {
			n += len(page)
		}
		return n
	}
	var walk func(any)
	walk = func(node any) {
		switch v := node.(type) {
		case map[string]any:
			if control, ok := v["control"].(string); ok && v["pages"] != nil {
				raw, _ := json.Marshal(v)
				var layout campaign.NativeDialogueLayout
				if json.Unmarshal(raw, &layout) == nil && layout.Validate() == nil {
					if old := best[control]; old == nil || rows(&layout) > rows(old) {
						copy := layout
						best[control] = &copy
					}
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var doc any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		walk(doc)
	}
	for _, control := range []string{"FFEC", "FFED", "FFEE", "FFEF"} {
		if best[control] == nil {
			t.Fatalf("no binding layout for %s", control)
		}
	}
	return best
}

func storyMaskTestGame(t *testing.T, locale string) *Game {
	t.Helper()
	ui, err := loadNativeClassUIAssets()
	if err != nil {
		t.Skipf("native class UI assets unavailable: %v", err)
	}
	g := &Game{localeID: locale, nativeClassUI: ui}
	if locale == "zh-Hant" {
		g.nativeBattleFont, g.nativeBattleGlyphs, err = loadNativeBattleNameAssets()
		if err != nil {
			t.Skipf("native glyph assets unavailable: %v", err)
		}
	} else if g.font = loadFont(); g.font == nil {
		t.Skip("display font unavailable")
	}
	return g
}

// prepareStoryMaskUtterance 以指定背景走正式 prepareNativeDialogueFrames；HD 美術只需
// 說話者的 80／480 圖存在，遮罩不讀圖素。
func prepareStoryMaskUtterance(t *testing.T, g *Game, layout *campaign.NativeDialogueLayout, base []byte, motionTargetY int, hd bool) {
	t.Helper()
	upper := layout.Control == "FFED" || layout.Control == "FFEF"
	g.dialog = []battle.DialogLine{{Speaker: 0, Upper: &upper, NativeDialogue: &battle.NativeDialogueLayout{
		SourceDAT: layout.SourceDAT, StringIndex: layout.StringIndex, Utterance: layout.Utterance,
		Control: layout.Control, Operand: layout.Operand, Pages: layout.Pages,
		GlyphPages: layout.GlyphPages, GlyphIDs: layout.GlyphIDs,
		MotionTargetY: motionTargetY, HasMotionTargetY: true,
	}}}
	g.modernStoryPortraits = nil
	if hd {
		g.modernStoryPortraits = &modernStoryPortraitSet{hasHD: true, portraits: map[int]*modernStoryPortraitFrame{
			0: {image: image.NewNRGBA(image.Rect(0, 0, 80, 80)), hdImage: image.NewNRGBA(image.Rect(0, 0, 480, 480))},
		}}
	}
	g.hasStoryNativeMapView = motionTargetY != 0
	g.storyNativeMapView.VisibleCursorX, g.storyNativeMapView.VisibleCursorY = 3, 2
	g.nativeMapVGA = append([]byte(nil), base...)
	if err := g.prepareNativeDialogueFrames(); err != nil {
		t.Fatalf("%s: %v", layout.Control, err)
	}
}

type storyMaskFrames struct {
	opening, closing, steady [][]byte
	pageOf                   []int
}

func collectStoryMaskFrames(t *testing.T, g *Game) storyMaskFrames {
	t.Helper()
	out := storyMaskFrames{opening: g.nativeDialogueOpening, closing: g.nativeDialogueClosing}
	for page, frames := range g.nativeDialogueProgressive {
		for _, frame := range frames {
			out.steady, out.pageOf = append(out.steady, frame), append(out.pageOf, page)
		}
		extra := [][]byte{g.nativeDialogueMouthOpen[page]}
		if g.localeID == "zh-Hant" {
			mouth, err := campaign.ComposeNativeStoryDialogueMouthFrame(frames[len(frames)-1], g.nativeDialoguePortraits[1], g.nativeDialogueLayout)
			if err != nil {
				t.Fatal(err)
			}
			extra = append(extra, mouth)
		}
		for _, stable := range [][]byte{frames[len(frames)-1], g.nativeDialogueMouthOpen[page]} {
			for phase := 0; phase < 2; phase++ {
				arrow, err := campaign.ComposeNativeStoryDialogueWaitArrow(stable, g.nativeClassUI.dialogue, g.nativeDialogueLayout, phase)
				if err != nil {
					t.Fatal(err)
				}
				extra = append(extra, arrow)
			}
		}
		for _, frame := range extra {
			out.steady, out.pageOf = append(out.steady, frame), append(out.pageOf, page)
		}
	}
	return out
}

// 遮罩必須與背景無關，遮罩外完全是背景，遮罩內完全是原生寫入：以兩張不同的
// 隨機背景各跑一次正式合成，三項都要成立。
func TestModernHDStoryDialogueMasksCoverEveryNativeWrite(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	layouts := storyMaskBindingLayouts(t)
	for _, locale := range []string{"zh-Hant", "en"} {
		for _, control := range []string{"FFEC", "FFED", "FFEE", "FFEF"} {
			for _, motion := range []int{0, 37} {
				layout := layouts[control]
				if locale != "zh-Hant" {
					copy := *layout
					copy.GlyphPages, copy.GlyphIDs = nil, nil
					copy.Pages = [][]string{{"Masked text", "two"}, {"next page"}}
					layout = &copy
				}
				name := locale + "/" + control
				if motion != 0 {
					name += "/motion"
				}
				t.Run(name, func(t *testing.T) {
					rng := rand.New(rand.NewSource(int64(len(control) + motion)))
					bases := [2][]byte{make([]byte, 320*200), make([]byte, 320*200)}
					for _, base := range bases {
						rng.Read(base)
					}
					var frames [2]storyMaskFrames
					var masks [2]*modernHDStoryDialogueMasks
					for run, base := range bases {
						g := storyMaskTestGame(t, locale)
						prepareStoryMaskUtterance(t, g, layout, base, motion, true)
						if g.nativeDialogueHDMasks == nil {
							t.Fatal("HD art did not freeze dialogue masks")
						}
						frames[run], masks[run] = collectStoryMaskFrames(t, g), g.nativeDialogueHDMasks
					}
					check := func(kind string, index int, mask []bool, a, b []byte) {
						for i := range mask {
							if mask[i] && a[i] != b[i] {
								t.Fatalf("%s %d: masked pixel %d depends on the background", kind, index, i)
							}
							if !mask[i] && (a[i] != bases[0][i] || b[i] != bases[1][i]) {
								t.Fatalf("%s %d: native write at %d is outside the mask", kind, index, i)
							}
						}
					}
					m := masks[0]
					if len(m.opening) != len(frames[0].opening) || len(m.closing) != len(frames[0].closing) ||
						len(m.pages) != len(layout.Pages) {
						t.Fatalf("mask counts %d/%d/%d", len(m.opening), len(m.closing), len(m.pages))
					}
					if motion != 0 && len(m.closing) <= 5 {
						t.Fatal("cursor tail was not exercised")
					}
					for index := range m.opening {
						check("opening", index, m.opening[index], frames[0].opening[index], frames[1].opening[index])
					}
					for index := range m.closing {
						check("closing", index, m.closing[index], frames[0].closing[index], frames[1].closing[index])
					}
					for index, page := range frames[0].pageOf {
						check("steady", index, m.pages[page], frames[0].steady[index], frames[1].steady[index])
					}
					for page := range m.pages {
						for i := range m.pages[page] {
							if m.pages[page][i] != masks[1].pages[page][i] {
								t.Fatalf("page %d mask depends on the background", page)
							}
						}
					}
				})
			}
		}
	}
}

func TestModernHDStoryDialogueMaskFallsBackToNativeFrame(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	layout := storyMaskBindingLayouts(t)["FFEE"]
	base := make([]byte, 320*200)
	rand.New(rand.NewSource(7)).Read(base)

	g := storyMaskTestGame(t, "zh-Hant")
	prepareStoryMaskUtterance(t, g, layout, base, 0, false)
	if g.nativeDialogueHDMasks != nil || g.modernHDStoryDialogueMask(campaign.NativeDialogueSteady, 0) != nil {
		t.Fatal("original art froze HD dialogue masks")
	}

	prepareStoryMaskUtterance(t, g, layout, base, 0, true)
	if g.modernHDStoryDialogueMask(campaign.NativeDialogueSteady, 0) == nil ||
		g.modernHDStoryDialogueMask(campaign.NativeDialogueOpening, 0) == nil {
		t.Fatal("HD art has no steady/opening mask")
	}
	g.dlgScrollT = 1
	if g.modernHDStoryDialogueMask(campaign.NativeDialogueSteady, 0) != nil {
		t.Fatal("page scroll did not fall back to the complete native frame")
	}
	g.dlgScrollT = 0
	if g.modernHDStoryDialogueMask(campaign.NativeDialogueOpening, len(g.nativeDialogueOpening)) != nil ||
		g.modernHDStoryDialogueMask(campaign.NativeDialogueClosing, -1) != nil {
		t.Fatal("out-of-range phase index produced a mask")
	}

	failing := func([]byte) ([][]byte, error) { return nil, errors.New("replay failed") }
	single := func(base []byte) ([][]byte, error) { return [][]byte{append([]byte(nil), base...)}, nil }
	if freezeModernHDStoryDialogueMasks(1, failing, single, func(base []byte, _ int) ([][]byte, error) { return single(base) }) != nil {
		t.Fatal("failed replay published masks")
	}
	calls := 0
	unstable := func(base []byte) ([][]byte, error) {
		calls++
		return make([][]byte, calls), nil
	}
	if freezeModernHDStoryDialogueMasks(1, single, unstable, func(base []byte, _ int) ([][]byte, error) { return single(base) }) != nil {
		t.Fatal("frame-count mismatch published masks")
	}
}

// 有 HD 世界時只上傳遮罩內的原生像素（遮罩外透明，HD 世界留在底下）；沒有 HD 世界
// 或沒有遮罩時照舊走完整原生幀。ReadPixels 需要 RunGame，這裡改看分支與上傳內容。
func TestModernHDStoryDialoguePresentKeepsHDWorld(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	layout := storyMaskBindingLayouts(t)["FFED"]
	base := make([]byte, 320*200)
	rand.New(rand.NewSource(11)).Read(base)
	g := storyMaskTestGame(t, "zh-Hant")
	prepareStoryMaskUtterance(t, g, layout, base, 0, true)
	frames := g.nativeDialogueProgressive[0]
	frame := frames[len(frames)-1]
	masks := g.nativeDialogueHDMasks
	mask := g.modernHDStoryDialogueMask(campaign.NativeDialogueSteady, 0)
	pixels, err := modernHDTimedDialoguePixels(frame, mask, g.nativeClassUI.palette)
	if err != nil {
		t.Fatal(err)
	}
	inside, outside := 0, 0
	for i, written := range mask {
		got := pixels.RGBAAt(i%320, i/320)
		if !written {
			outside++
			if got.A != 0 {
				t.Fatalf("pixel %d outside the mask covers the HD world", i)
			}
			continue
		}
		inside++
		r, gg, b, _ := g.nativeClassUI.palette[frame[i]].RGBA()
		if got.R != byte(r>>8) || got.G != byte(gg>>8) || got.B != byte(b>>8) || got.A != 255 {
			t.Fatalf("pixel %d inside the mask lost its native write", i)
		}
	}
	if inside == 0 || outside == 0 {
		t.Fatal("mask has no inside/outside sample")
	}
	for _, c := range []struct {
		world, masked, wantMasked bool
	}{{true, true, true}, {false, true, false}, {true, false, false}} {
		g.modernHDWorldVisible, g.nativeDialogueHDMasks = c.world, nil
		if c.masked {
			g.nativeDialogueHDMasks = masks
		}
		g.modernHDMapGPU = &modernHDMapGPU{}
		if err := g.presentNativeStoryDialogueFrame(ebiten.NewImage(640, 400), frame, campaign.NativeDialogueSteady, 0, nil, image.Rectangle{}); err != nil {
			t.Fatal(err)
		}
		if (g.modernHDMapGPU.Dialogue != nil) != c.wantMasked {
			t.Fatalf("world=%v masks=%v took the wrong presentation path", c.world, c.masked)
		}
	}
}
