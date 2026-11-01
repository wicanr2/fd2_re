package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

func countIconAssetForTest(t *testing.T) (modernThemeAsset, string, modernThemeCatalog) {
	t.Helper()
	catalog, pack := os.Getenv("FD2_TEST_FACILITY_CATALOG"), os.Getenv("FD2_TEST_FACILITY_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit source-bound UI catalog and candidate pack required")
	}
	raw, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var data modernThemeCatalog
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	for _, asset := range data.Assets {
		if asset.Role == "native_ui_embedded_icon_hd" {
			return asset, pack, data
		}
	}
	t.Fatal("count symbol missing from explicit catalog")
	return modernThemeAsset{}, "", data
}

func TestModernHDUIEmbeddedIconLoaderContract(t *testing.T) {
	asset, pack, data := countIconAssetForTest(t)
	set := &modernStoryPortraitSet{}
	if err := loadModernUIEmbeddedIcon(set, asset, pack); err != nil {
		t.Fatal(err)
	}
	if icon := set.uiEmbeddedIcons[137]; icon == nil || icon.source != image.Rect(20, 22, 44, 40) || icon.picture.Bounds() != image.Rect(0, 0, 144, 108) {
		t.Fatal("source rectangle/shape not retained")
	}
	if err := loadModernUIEmbeddedIcon(set, asset, pack); err == nil {
		t.Fatal("duplicate original source accepted")
	}
	for _, fault := range []string{"role", "resource", "metadata", "source-frame", "source-hash", "rect", "size", "contract", "alpha", "path", "master-proof", "missing-png", "png-size", "opaque-png", "blank-png", "png-hash"} {
		item := asset
		item.NativeLMIFrames = append([]modernLMIHDDescriptor(nil), asset.NativeLMIFrames...)
		item.NativeSourceRect = append([]int(nil), asset.NativeSourceRect...)
		item.SourceSHA256 = append([]string(nil), asset.SourceSHA256...)
		root := pack
		switch fault {
		case "role":
			item.Role = "native_ui_icon_hd"
		case "resource":
			item.SourceResource = 6
		case "metadata":
			item.SourceMetadataSHA256 = "bad"
		case "source-frame":
			item.NativeLMIFrames[0].Index = 136
		case "source-hash":
			item.NativeLMIFrames[0].PixelsSHA256 = "bad"
		case "rect":
			item.NativeSourceRect[0]++
		case "size":
			item.Width--
		case "contract":
			item.ConsumerContract = "native_raw_ui_icon_hd6_v1"
		case "alpha":
			item.AlphaContract = "opaque"
		case "path":
			item.File = "../bad.png"
		case "master-proof":
			item.SourceSHA256[0] = "bad"
		case "missing-png":
			item.File = "missing-count-symbol.png"
		case "png-hash":
			item.SHA256 = "bad"
		case "png-size", "opaque-png", "blank-png":
			root = t.TempDir()
			item.File = "icon.png"
			bounds := image.Rect(0, 0, 144, 108)
			if fault == "png-size" {
				bounds.Max.X = 143
			}
			picture := image.NewRGBA(bounds)
			if fault != "blank-png" {
				draw.Draw(picture, picture.Bounds(), image.NewUniform(color.RGBA{10, 20, 30, 255}), image.Point{}, draw.Src)
			}
			var b bytes.Buffer
			if err := png.Encode(&b, picture); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, item.File), b.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(b.Bytes())
			item.SHA256 = hex.EncodeToString(sum[:])
		}
		got := &modernStoryPortraitSet{}
		if err := loadModernUIEmbeddedIcon(got, item, root); err == nil || len(got.uiEmbeddedIcons) != 0 {
			t.Fatal("invalid source published", fault, err)
		}
	}
	// Keep the existing required story portrait pair in the minimal old UI catalog.
	var old []modernThemeAsset
	for _, entry := range data.Assets {
		if entry.Role == "native_ui_icon_hd" || (entry.Role == "story_portrait_frame" || entry.Role == "story_portrait_hd_frame") && entry.SpeakerID == 0 {
			old = append(old, entry)
		}
	}
	if len(old) != 12 {
		t.Fatal("old UI group missing")
	}
	data.Assets = old
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "legacy-catalog.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	legacy, err := loadModernStoryPortraitSet(path, pack)
	if err != nil || len(legacy.uiEmbeddedIcons) != 0 || len(legacy.uiPanelIcons) != 10 {
		t.Fatal("legacy optional-role absence rejected", err)
	}
}

func verifyPreparationCountIconCPU(t *testing.T, g *Game, set *modernStoryPortraitSet) {
	t.Helper()
	saved := *g
	defer func() { *g = saved }()
	g.nativeClassUIJob = nil
	preparationHDTestPhase(g, "roster")
	g.modernStoryPortraits, g.modernThemeCache, g.interfaceMode = set, set, playerInterfaceHD
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	g.Draw(screen)
	h := g.prepareModernHDPreparationCounts(g.nativePreparationUI)
	if h == nil || h.head == nil || h.head != set.uiEmbeddedIcons[137] {
		t.Fatal("new count material not published")
	}
	for _, art := range []*modernStoryPortraitSet{nil, set} {
		g.modernStoryPortraits = art
		g.Draw(screen)
		if next := g.prepareModernHDPreparationCounts(g.nativePreparationUI); next != h {
			t.Fatal("F2 changed UI count symbol")
		}
	}
	legacy := *set
	legacy.uiEmbeddedIcons = nil
	g.modernStoryPortraits, g.modernThemeCache = &legacy, &legacy
	g.Draw(screen)
	old := g.prepareModernHDPreparationCounts(g.nativePreparationUI)
	if old == nil || old.head != nil || old == h {
		t.Fatal("old role absence kept stale head")
	}
	for _, rect := range []image.Rectangle{image.Rect(120, 132, 264, 240), image.Rect(120, 360, 264, 468)} {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				if old.labels.picture.RGBAAt(x, y).A != 0 {
					t.Fatal("old native symbol erased")
				}
			}
		}
	}
	for _, fault := range []string{"pixels", "descriptor", "rect", "shape"} {
		bad := *set
		bad.uiEmbeddedIcons = map[int]*modernUIEmbeddedIcon{}
		icon := *set.uiEmbeddedIcons[137]
		bad.uiEmbeddedIcons[137] = &icon
		switch fault {
		case "pixels":
			icon.digest[0] ^= 255
		case "descriptor":
			icon.native.Index = 136
		case "rect":
			icon.source = icon.source.Add(image.Pt(1, 0))
		case "shape":
			icon.picture = image.NewRGBA(image.Rect(0, 0, 143, 108))
		}
		g.modernStoryPortraits, g.modernThemeCache = &bad, &bad
		g.Draw(screen)
		if g.prepareModernHDPreparationCounts(g.nativePreparationUI) != nil {
			t.Fatal("changed head source reused old material", fault)
		}
		for _, panel := range g.modernHDText.panels {
			if panel.rect == image.Rect(5, 7, 91, 93) {
				t.Fatal("invalid head material stayed visible", fault)
			}
		}
	}
}

func TestModernHDPreparationCountIconRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PREPARATION_COUNT_ICON_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	runModernHDPreparationInterfaceMatrix(t, true, true, true)
}

func appendPreparationLegacyCountCases(p *facilityHDProbe, set *modernStoryPortraitSet) {
	g := p.g
	legacy := *set
	legacy.uiEmbeddedIcons = nil
	camp, town, source := g.camp, g.prepTownSource, append([]byte(nil), g.prepPromptSource...)
	for style := 0; style < 4; style++ {
		style := style
		p.captureCases[len(p.cases)] = true
		p.cases = append(p.cases, func() {
			g.camp, g.prepTownSource, g.prepPromptSource = camp, town, source
			g.nativeClassUIJob = nil
			preparationHDTestPhase(g, "roster")
			g.prepLimit, g.prepSel = 30, 0
			g.modernThemeCache = &legacy
			g.modernStoryPortraits = nil
			if style >= 2 {
				g.modernStoryPortraits = &legacy
			}
			g.interfaceMode = playerInterfaceOriginal
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
		})
	}
}

func verifyPreparationCountIconGPU(p *facilityHDProbe, pixels []byte) error {
	g := p.g
	if g.interfaceMode == playerInterfaceOriginal {
		return nil
	}
	h := g.prepareModernHDPreparationCounts(g.nativePreparationUI)
	if h == nil {
		return fmt.Errorf("case%d count material missing", p.index)
	}
	world := []image.Point{image.Pt(25*6, 29*6), image.Pt(25*6, 67*6)}
	var expected *image.RGBA
	var native []byte
	if h.head != nil {
		index := g.nativePreparationUI.roster.UpperLeft.Indexed[4*86+4]
		bg := color.RGBAModel.Convert(g.nativeClassUI.palette[index]).(color.RGBA)
		expected = image.NewRGBA(image.Rect(0, 0, 144, 108))
		draw.Draw(expected, expected.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
		draw.Draw(expected, expected.Bounds(), h.head.picture, image.Point{}, draw.Over)
	} else {
		var ok bool
		native, ok = g.composeNativePreparationStableFrame()
		if !ok {
			return fmt.Errorf("legacy native count control missing")
		}
	}
	checked := 0
	for _, origin := range world {
		for y := 0; y < 108; y++ {
			for x := 0; x < 144; x++ {
				var want color.RGBA
				if expected != nil {
					want = expected.RGBAAt(x, y)
				} else {
					value := native[(origin.Y+y)/6*320+(origin.X+x)/6]
					want = color.RGBAModel.Convert(g.nativeClassUI.palette[value]).(color.RGBA)
				}
				at := ((origin.Y+y+72)*1920 + origin.X + x) * 4
				if !bytes.Equal(pixels[at:at+4], []byte{want.R, want.G, want.B, want.A}) {
					return fmt.Errorf("case%d count icon source pixel %d,%d", p.index, origin.X+x, origin.Y+y)
				}
				checked++
			}
		}
	}
	last := p.checks[len(p.checks)-1]
	if expected != nil {
		last["count_icon_pixels"] = checked
	} else {
		last["legacy_count_icon_pixels"] = checked
	}
	return nil
}
