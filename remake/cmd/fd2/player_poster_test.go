package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/text"
)

func posterTestCatalog(t *testing.T) (string, string, modernThemeCatalog) {
	t.Helper()
	root := t.TempDir()
	c := modernThemeCatalog{SchemaVersion: 1, ThemeID: modernHandpaintedThemeID}
	for i, digest := range []string{"0899058cd5fcb458305e1b6db6fe5880e10e9193a082ec2aa3616aca659b5308", "55285013e44390c187ae85535ccdab27ab8f9040485559cc6800a94be343df3d"} {
		name := "poster-set1-composite.png"
		source := "fd2-hd2-poster-set1-source.png"
		if i == 1 {
			name, source = "poster-set2-composite.png", "fd2-hd2-poster-set2-source.png"
		}
		path, err := filepath.Abs(filepath.Join("..", "..", "..", "docs", "figures", source))
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		actual := sha256.Sum256(raw)
		if hex.EncodeToString(actual[:]) != digest {
			t.Fatal("approved poster source changed")
		}
		if err := os.Symlink(path, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
		c.Assets = append(c.Assets, modernThemeAsset{Role: playerPosterAssetRole, Status: "runtime_candidate", File: name,
			Width: 1672, Height: 941, SHA256: digest, Frame: i, ConsumerContract: "composite_poster_sides_v1",
			LayoutPolicy: "native_game_center_sides_v1", AlphaContract: "opaque"})
	}
	path := filepath.Join(root, "catalog.json")
	writePosterTestCatalog(t, path, c)
	return path, root, c
}

func writePosterTestCatalog(t *testing.T, path string, c modernThemeCatalog) {
	t.Helper()
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestPlayerPosterApprovedPixelsAndAtomicSourceRejection(t *testing.T) {
	path, root, catalog := posterTestCatalog(t)
	pair, err := loadPlayerPosterPair(path, root)
	if err != nil {
		t.Fatal(err)
	}
	if formal := os.Getenv("FD2_TEST_FACILITY_CATALOG"); formal != "" {
		admitted, err := loadPlayerPosterPair(formal, os.Getenv("FD2_TEST_FACILITY_PACK"))
		if err != nil {
			t.Fatal("registered poster pair", err)
		}
		for i := range pair.Sides {
			for side := range pair.Sides[i] {
				if !bytes.Equal(pair.Sides[i][side].Pix, admitted.Sides[i][side].Pix) {
					t.Fatal("registered poster differs from approved source", i, side)
				}
			}
		}
	}
	for i, a := range catalog.Assets {
		file, err := os.Open(filepath.Join(root, a.File))
		if err != nil {
			t.Fatal(err)
		}
		source, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		for side, x0 := range []int{0, 1376} {
			got := pair.Sides[i][side]
			if got.Bounds() != image.Rect(0, 0, 296, 900) {
				t.Fatal("poster contains baked footer or wrong crop")
			}
			for y := 0; y < 900; y++ {
				for x := 0; x < 296; x++ {
					if color.RGBAModel.Convert(source.At(x0+x, y)) != got.RGBAAt(x, y) {
						t.Fatal("selected poster pixels were edited", i, side, x, y)
					}
				}
			}
		}
	}
	for _, fault := range []string{"missing", "sha", "path", "size", "format-size", "alpha", "contract", "layout", "pair", "duplicate", "index", "schema", "identity"} {
		t.Run(fault, func(t *testing.T) {
			c := catalog
			c.Assets = append([]modernThemeAsset(nil), catalog.Assets...)
			a := &c.Assets[1]
			switch fault {
			case "missing":
				a.File = "missing.png"
			case "sha":
				a.SHA256 = "wrong"
			case "path":
				a.File = "../" + a.File
			case "size":
				a.Width--
			case "contract":
				a.ConsumerContract = "guessed"
			case "layout":
				a.LayoutPolicy = "guessed"
			case "pair":
				c.Assets = c.Assets[:1]
			case "duplicate":
				c.Assets = append(c.Assets, c.Assets[0])
			case "index":
				a.Frame = 2
			case "schema":
				c.SchemaVersion++
			case "identity":
				c.ThemeID = "wrong"
			case "format-size", "alpha":
				picture := image.NewNRGBA(image.Rect(0, 0, 1672, 941))
				for i := 3; i < len(picture.Pix); i += 4 {
					picture.Pix[i] = 255
				}
				if fault == "alpha" {
					picture.Pix[3] = 254
				} else {
					picture = image.NewNRGBA(image.Rect(0, 0, 2, 2))
				}
				var raw bytes.Buffer
				if err := png.Encode(&raw, picture); err != nil {
					t.Fatal(err)
				}
				a.File = fault + ".png"
				digest := sha256.Sum256(raw.Bytes())
				a.SHA256 = hex.EncodeToString(digest[:])
				if err := os.WriteFile(filepath.Join(root, a.File), raw.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
			writePosterTestCatalog(t, path, c)
			if got, err := loadPlayerPosterPair(path, root); err == nil || got != nil {
				t.Fatal("bad poster pair was partially published", fault, err)
			}
		})
	}
}

func TestPlayerPosterGeometryInputAndBlocking(t *testing.T) {
	g := &Game{}
	p := &modernHDPresentation{Game: g, toolbar: true, floatingToolbar: true}
	p.poster.settings = defaultPlayerPosters()
	p.poster.settings.Set, p.poster.pair = "hd", &playerPosterPair{}
	for _, fixed := range []bool{false, true} {
		p.floatingToolbar = !fixed
		w, h := p.Layout(1, 1)
		if w != 904 || h != 400+p.toolbarOffset() || p.presentationHeight(1920) != int(1920*float64(h)/float64(w)+.5) {
			t.Fatal("poster stretched the logical game", w, h)
		}
		g.playerViewportLeft, g.playerViewportTop = p.posterOffset(), p.toolbarOffset()
		for _, xy := range []image.Point{{0, 0}, {17, 72}, {639, 399}} {
			x, y, ok := g.playerTouchCoordinates(xy.X+132, xy.Y+p.toolbarOffset())
			if !ok || x != xy.X || y != xy.Y {
				t.Fatal("poster changed the selected game point", xy, x, y, ok)
			}
		}
		for _, x := range []int{0, 131, 772, 903} {
			if _, _, ok := g.playerTouchCoordinates(x, 200); ok {
				t.Fatal("poster touch entered the game")
			}
		}
		g.playerOverlayBlock = playerPosterMenu
		if _, _, ok := g.playerTouchCoordinates(132+400, 120); ok {
			t.Fatal("menu touch entered the game")
		}
		g.playerOverlayBlock = image.Rectangle{}
	}
	for _, blocked := range []func(){func() { g.helpVisible = true }, func() { g.transientUI = true }, func() { g.startupBlocked = true }} {
		*g = Game{}
		blocked()
		p.queueToolbarPoint(image.Pt(505, 12))
		if p.poster.menuOpen || p.poster.consumeInput {
			t.Fatal("blocked poster request was retained")
		}
	}
}

func TestPlayerPosterMaskFadeAndSourceRemainIndependent(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 296, 900))
	for i := 0; i < len(source.Pix); i += 4 {
		copy(source.Pix[i:i+4], []byte{40, 70, 110, 255})
	}
	before := append([]byte(nil), source.Pix...)
	for _, side := range []int{0, 1} {
		masked := maskPlayerPoster(source, side, 48)
		for distance, alpha := range map[int]byte{0: 0, 24: 64, 48: 255, 100: 255} {
			x := distance
			if side == 0 {
				x = 295 - distance
			}
			if got := masked.NRGBAAt(x, 450); got != (color.NRGBA{40, 70, 110, alpha}) {
				t.Fatal("poster gradient changed color or prototype alpha", side, distance, got)
			}
		}
	}
	if !bytes.Equal(before, source.Pix) {
		t.Fatal("poster gradient edited approved source")
	}
	start := time.Unix(100, 0)
	now := start.Add(300 * time.Millisecond)
	p := &modernHDPresentation{Game: &Game{playerSpeed: 4}}
	p.poster.settings, p.poster.started, p.poster.clock = defaultPlayerPosters(), start, func() time.Time { return now }
	if p.posterAlpha() != .5 {
		t.Fatal("game speed changed real-time poster fade")
	}
	for _, item := range []struct {
		delta time.Duration
		alpha float64
	}{{-1, 0}, {0, 0}, {600 * time.Millisecond, 1}, {time.Second, 1}} {
		now = start.Add(item.delta)
		if p.posterAlpha() != item.alpha {
			t.Fatal("poster fade outside bounds")
		}
	}
	for x, want := range map[int]int{0: 0, 380: 0, 499: 48, 619: 96, 999: 96} {
		if playerPosterEdgeAt(x) != want {
			t.Fatal("slider boundary changed", x)
		}
	}
}

func TestPlayerPosterPreferencesPreserveOtherSettingsAndRejectAtomically(t *testing.T) {
	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = oldCache })
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	poster := playerPosterSettings{Set: "hd", Edge: 0, FadeMS: 0}
	saveSettings(settings{BGMSource: "fm", LocaleID: "zh-Hant", FontScale: 1.25, GameSpeed: 3, InterfaceMode: "original", ThemeID: modernHandpaintedThemeID, Poster: &poster})
	g := &Game{bgmSource: "fm", localeID: "zh-Hant", fontScale: 1.25, interfaceMode: "original", modernStoryPortraits: &modernStoryPortraitSet{hasHD: true}}
	attachOfficialLocale(t, g)
	g.cycleBGMSource()
	g.cycleLocale()
	g.saveDisplaySettings()
	s := loadSettings()
	if *s.Poster != poster || s.GameSpeed != 3 || s.InterfaceMode != "original" || s.ThemeID != modernHandpaintedThemeID || s.FontScale != 1.25 {
		t.Fatal("another control erased poster settings", s)
	}
	raw, _ := os.ReadFile(settingsPath())
	p := &modernHDPresentation{Game: g}
	path, root, catalog := posterTestCatalog(t)
	p.poster.catalogPath, p.poster.packRoot = path, root
	loaded, err := loadPlayerPosterPair(path, root)
	if err != nil {
		t.Fatal(err)
	}
	p.poster.pair = loaded
	p.poster.settings = poster
	pair := p.poster.pair
	broken := catalog
	broken.Assets = append([]modernThemeAsset(nil), catalog.Assets...)
	broken.Assets[1].SHA256 = "bad"
	writePosterTestCatalog(t, path, broken)
	if err := p.applyPosterMode("classic", true); err == nil || p.poster.settings != poster || p.poster.pair != pair {
		t.Fatal("warm cache bypassed missing source or published partial settings")
	}
	actual, _ := os.ReadFile(settingsPath())
	if !bytes.Equal(raw, actual) {
		t.Fatal("rejected poster switch changed settings")
	}
	writePosterTestCatalog(t, path, catalog)
	if err := p.applyPosterMode("classic", true); err != nil || loadSettings().Poster.Set != "classic" || loadSettings().GameSpeed != 3 {
		t.Fatal("valid poster selection failed or erased another preference", err)
	}
	currentPair := p.poster.pair
	raw, _ = os.ReadFile(settingsPath())
	if err := os.Remove(filepath.Join(root, catalog.Assets[0].File)); err != nil {
		t.Fatal(err)
	}
	if err := p.applyPosterMode("hd", true); err == nil || p.poster.pair != currentPair || p.poster.settings.Set != "classic" {
		t.Fatal("warm poster cache bypassed missing source")
	}
	actual, _ = os.ReadFile(settingsPath())
	if !bytes.Equal(raw, actual) {
		t.Fatal("missing source changed saved settings")
	}
	if normalizePlayerPosters(nil) != defaultPlayerPosters() || normalizePlayerPosters(&playerPosterSettings{Set: "bad", Edge: -1, FadeMS: 9}) != defaultPlayerPosters() {
		t.Fatal("legacy or invalid poster settings changed defaults")
	}
}

func TestPlayerPosterMenuLabelsFitAllLocales(t *testing.T) {
	f := loadFont()
	if f == nil {
		t.Fatal("Noto font required")
	}
	for _, locale := range localeIDs {
		p := &modernHDPresentation{Game: &Game{font: f, localeID: locale}}
		labels := p.posterLabels()
		local := f
		if regional := f.hdFonts[locale]; regional != nil {
			local = regional
		}
		for i, label := range labels.sets {
			rect := playerPosterChoices[i]
			face, caption, at := playerToolbarLabelLayout(local, label, rect)
			ink := text.BoundString(face, caption).Add(at)
			if caption != label || !ink.In(image.Rect(rect.Min.X*3+12, rect.Min.Y*3+6, rect.Max.X*3-12, rect.Max.Y*3-6)) {
				t.Fatal("poster choice was truncated or left its button", locale, label, ink)
			}
		}
	}
}
