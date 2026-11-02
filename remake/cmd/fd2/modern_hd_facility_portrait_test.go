package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

func TestModernHDFacilityPortraitSourceAdapter(t *testing.T) {
	for _, speaker := range []int{129, 131} {
		frames, err := loadNativeSeparatedPortrait(speaker)
		if err != nil || !nativeFacilityPortraitFramesMatch(speaker, frames) {
			t.Fatal("original typed source differs", speaker, err)
		}
		offset := campaign.NativeFacilityPortraitOffset(speaker)
		if offset != map[int]int{129: 0x6ab, 131: 0x576}[speaker] {
			t.Fatal("original facility position differs", speaker)
		}
	}
}

func TestModernHDFacilityPortraitVisibilityAndIndependentUI(t *testing.T) {
	for _, speaker := range []int{129, 131} {
		sources, err := loadNativeSeparatedPortrait(speaker)
		if err != nil {
			t.Fatal(err)
		}
		portrait := &modernStoryPortraitFrame{image: image.NewRGBA(image.Rect(0, 0, 80, 80)), hdImage: image.NewRGBA(image.Rect(0, 0, 480, 480))}
		set := &modernStoryPortraitSet{portraits: map[int]*modernStoryPortraitFrame{speaker: portrait}}
		offset := map[int]int{129: 0x6ab, 131: 0x576}[speaker]
		for mouth, source := range sources {
			frame := bytes.Repeat([]byte{73}, 64000)
			if err := source.BlitAtOffset(frame, 320, offset); err != nil {
				t.Fatal(err)
			}
			before := append([]byte(nil), frame...)
			for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD} {
				g := &Game{modernStoryPortraits: set, interfaceMode: mode}
				if !g.appendModernHDFacilityPortraitFor(frame, speaker, sources) || len(g.modernHDPanelPortraits) != 1 {
					t.Fatal("visible source missing", speaker, mouth, mode)
				}
				x, y := offset%320, offset/320
				overlay := g.modernHDPanelPortraits[0]
				if overlay.portrait != portrait || overlay.rect != image.Rect(x*2, y*2, (x+80)*2, (y+80)*2) || overlay.clip != overlay.rect {
					t.Fatal("source or original position changed")
				}
				g.modernStoryPortraits = nil
				g.modernHDPanelPortraits = nil
				if g.appendModernHDFacilityPortraitFor(frame, speaker, sources) || len(g.modernHDPanelPortraits) != 0 {
					t.Fatal("portrait followed UI instead of art")
				}
			}
			if !bytes.Equal(frame, before) {
				t.Fatal("overlay changed indexed source")
			}
		}
		frame := make([]byte, 64000)
		if err := sources[0].BlitAtOffset(frame, 320, offset); err != nil {
			t.Fatal(err)
		}
		for _, fault := range []string{"covered", "partial-frame", "source-pixels", "source-size", "source-count", "missing-portrait", "missing-hd", "hd-size", "job", "unknown"} {
			pixels := append([]byte(nil), frame...)
			originals := append([]dato.Frame(nil), sources...)
			copyPortrait := *portrait
			g := &Game{modernStoryPortraits: &modernStoryPortraitSet{portraits: map[int]*modernStoryPortraitFrame{speaker: &copyPortrait}}}
			id := speaker
			switch fault {
			case "covered":
				pixels[offset+10*320+10] ^= 1
			case "partial-frame":
				pixels = pixels[:1]
			case "source-pixels":
				originals[3].Pixels = append([]byte(nil), sources[3].Pixels...)
				originals[3].Pixels[0] ^= 1
			case "source-size":
				originals[0].Width--
			case "source-count":
				originals = originals[:1]
			case "missing-portrait":
				delete(g.modernStoryPortraits.portraits, speaker)
			case "missing-hd":
				copyPortrait.hdImage = nil
			case "hd-size":
				copyPortrait.hdImage = image.NewRGBA(image.Rect(0, 0, 80, 80))
			case "job":
				g.nativeClassUIJob = &nativeClassUIJob{}
			case "unknown":
				id = 128
			}
			if g.appendModernHDFacilityPortraitFor(pixels, id, originals) || len(g.modernHDPanelPortraits) != 0 {
				t.Fatal("unsafe source published", speaker, fault)
			}
		}
	}
}

func TestModernHDFacilityPortraitRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_FACILITY_PORTRAITS_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	var err error
	g.nativeHotelUI, err = loadNativeHotelUIAssets(g.nativeClassUI)
	if err != nil {
		t.Fatal(err)
	}
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	legacy := *set
	legacy.portraits = make(map[int]*modernStoryPortraitFrame)
	for id, portrait := range set.portraits {
		if id != 129 && id != 131 {
			legacy.portraits[id] = portrait
		}
	}
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true},
		captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		g := p.g
		id := 129
		var frame []byte
		var ok bool
		if g.camp.Node().Type == "hotel" {
			frame, ok = g.composeNativeHotelFrame(0)
		} else {
			id = 131
			frame, ok = g.composeNativeChurchMenuFrame()
		}
		if !ok {
			return fmt.Errorf("case%d native source missing", p.index)
		}
		offset := map[int]int{129: 0x6ab, 131: 0x576}[id]
		x, y := offset%320, offset/320
		portrait := (*modernStoryPortraitFrame)(nil)
		if g.modernStoryPortraits != nil {
			portrait = g.modernStoryPortraits.portraits[id]
		}
		if (len(g.modernHDPanelPortraits) == 1) != (portrait != nil) {
			return fmt.Errorf("case%d portrait not tied to own source/art", p.index)
		}
		base := make([]byte, 640*424*4)
		offscreen.ReadPixels(base)
		activeScene := g.modernHDScene
		if (activeScene != nil) != (portrait != nil) {
			return fmt.Errorf("case%d source/art scene admission differs", p.index)
		}
		background := 0
		if activeScene != nil {
			for py := 0; py < 1200; py++ {
				for px := 0; px < 1920; px++ {
					atBase := ((py/3+24)*640 + px/3) * 4
					if base[atBase+3] != 0 {
						continue
					}
					inText := false
					if g.modernHDText != nil {
						for _, run := range g.modernHDText.runs {
							if image.Pt(px/3, py/3).In(run.clip) {
								inText = true
								break
							}
						}
					}
					if inText {
						continue
					}
					want := color.RGBAModel.Convert(activeScene.scene.image.At(px, py)).(color.RGBA)
					at := ((py+72)*1920 + px) * 4
					if !bytes.Equal(pixels[at:at+4], []byte{want.R, want.G, want.B, 255}) {
						return fmt.Errorf("case%d full scene source %d,%d", p.index, px, py)
					}
					background++
				}
			}
			if background < 100000 {
				return fmt.Errorf("case%d too little exposed full scene", p.index)
			}
		}
		checked, native := 0, 0
		for sy := 0; sy < 480; sy++ {
			for sx := 0; sx < 480; sx++ {
				var want color.RGBA
				if portrait != nil {
					want = color.RGBAModel.Convert(portrait.hdImage.At(sx, sy)).(color.RGBA)
				} else {
					want = color.RGBAModel.Convert(g.nativeClassUI.palette[frame[(y+sy/6)*320+x+sx/6]]).(color.RGBA)
				}
				at := (((y*6+sy)+72)*1920 + x*6 + sx) * 4
				if !bytes.Equal(pixels[at:at+4], []byte{want.R, want.G, want.B, 255}) {
					return fmt.Errorf("case%d portrait%d pixel %d,%d", p.index, id, sx, sy)
				}
				checked++
			}
		}
		if g.interfaceMode == playerInterfaceOriginal {
			head := image.Rect(x*2, y*2, (x+80)*2, (y+80)*2)
			for ly := 0; ly < 400; ly++ {
				for lx := 0; lx < 640; lx++ {
					want := color.RGBAModel.Convert(g.nativeClassUI.palette[frame[ly/2*320+lx/2]]).(color.RGBA)
					at := ((ly+24)*640 + lx) * 4
					if base[at+3] == 0 {
						continue
					}
					if !bytes.Equal(base[at:at+4], []byte{want.R, want.G, want.B, 255}) {
						return fmt.Errorf("case%d original underlay changed %d,%d", p.index, lx, ly)
					}
					if portrait == nil || !image.Pt(lx, ly).In(head) {
						dest := (((ly+24)*3+1)*1920 + lx*3 + 1) * 4
						if !bytes.Equal(pixels[dest:dest+4], base[at:at+4]) {
							return fmt.Errorf("case%d foreground outside source %d,%d", p.index, lx, ly)
						}
						native++
					}
				}
			}
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "speaker": id, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "hd_source_available": portrait != nil, "portrait_pixels": checked, "native_final_pixels": native, "exposed_scene_pixels": background, "full_scene": activeScene != nil})
		return nil
	}
	add := func(owner, mode string, selection, style int, theme *modernStoryPortraitSet) {
		if style == 3 {
			p.captureCases[len(p.cases)] = true
		}
		p.cases = append(p.cases, func() {
			g.camp = campaign.NewRunner(&campaign.Campaign{Start: "facility", Nodes: map[string]*campaign.Node{"facility": {Type: owner}}})
			g.nativeClassUIJob, g.nativeChurchUIJob = nil, nil
			g.nativeShopUIJob = nil
			g.nativeShopMode, g.nativeHotelMode, g.churchMode = "", "", ""
			g.modernThemeCache = theme
			g.modernStoryPortraits = nil
			if style >= 2 {
				g.modernStoryPortraits = theme
			}
			g.interfaceMode = playerInterfaceOriginal
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
			if owner == "hotel" {
				g.nativeHotelMode = mode
				g.hotelSel = selection
			} else {
				g.churchMode = "menu"
				g.churchSel = selection
				g.nativeChurchTextIndex = 586
			}
		})
	}
	for _, owner := range []string{"hotel", "church"} {
		for selection := 0; selection < 4; selection++ {
			for style := 0; style < 4; style++ {
				add(owner, "menu", selection, style, set)
			}
		}
	}
	for _, mode := range []string{"saved", "loaded", "load_blocked"} {
		for style := 0; style < 4; style++ {
			add("hotel", mode, 0, style, set)
		}
	}
	for _, owner := range []string{"hotel", "church"} {
		for _, style := range []int{2, 3} {
			add(owner, "menu", 0, style, &legacy)
		}
	}
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != 48 {
		t.Fatal("incomplete facility matrix", len(p.checks))
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-FACILITY-SCENE-PORTRAITS-GPU", "scope": "48 full Game.Draw cases; paired full hotel/church scenes, exact portrait crops, native foreground writes, four styles and missing-pair whole-native fallback; no new original E2", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestModernHDFacilityFullSourceExport(t *testing.T) {
	out := os.Getenv("FD2_FACILITY_FULL_SOURCE_OUT")
	if out == "" {
		t.Skip("explicit decoded-source export required")
	}
	if info, err := os.Stat(out); err != nil || !info.IsDir() {
		t.Fatal("existing source output directory required", err)
	}
	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	hotel, err := loadNativeHotelUIAssets(shared)
	if err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	for _, speaker := range []int{129, 131} {
		source := shared.background
		portraits := shared.portraits
		name := "church"
		if speaker == 129 {
			source = hotel.assets.Background
			portraits = hotel.portraits
			name = "hotel"
		}
		if len(source) != 64000 || !nativeFacilityPortraitFramesMatch(speaker, portraits) {
			t.Fatal("original source differs", speaker)
		}
		frame := append([]byte(nil), source...)
		offset := campaign.NativeFacilityPortraitOffset(speaker)
		if err := portraits[0].BlitAtOffset(frame, 320, offset); err != nil {
			t.Fatal(err)
		}
		picture := image.NewPaletted(image.Rect(0, 0, 320, 200), shared.palette)
		copy(picture.Pix, frame)
		path := filepath.Join(out, "source-"+name+"-composed.png")
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, picture); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, map[string]any{"facility": name, "speaker": speaker, "resource": map[int]int{129: 13, 131: 14}[speaker],
			"file": path, "sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(frame)),
			"background_indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(source)),
			"native_portrait_rect":      []int{offset % 320, offset / 320, offset%320 + 80, offset/320 + 80}})
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-DECODED-FACILITY-SOURCE-COMPOSITION",
		"scope":          "fixed separated FDOTHER background plus original DATO m0 at proven position; no original runtime/oracle or player reachability claim",
		"source_archive": "FDOTHER.DAT", "source_sha256": "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce", "sources": records}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "full-scene-source.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestModernHDFacilityScenePairAndMarkerMask(t *testing.T) {
	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	hotel, err := loadNativeHotelUIAssets(shared)
	if err != nil {
		t.Fatal(err)
	}
	set := &modernStoryPortraitSet{portraits: make(map[int]*modernStoryPortraitFrame), facilityScenes: make(map[int]*modernFacilityScene)}
	for resource, speaker := range map[int]int{13: 129, 14: 131} {
		scene := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		draw.Draw(scene, scene.Bounds(), image.NewUniform(color.RGBA{R: byte(resource), G: 79, B: 143, A: 255}), image.Point{}, draw.Src)
		offset := map[int]int{13: 0x6ab, 14: 0x576}[resource]
		x, y := offset%320*6, offset/320*6
		head := image.NewRGBA(image.Rect(0, 0, 480, 480))
		draw.Draw(head, head.Bounds(), scene, image.Pt(x, y), draw.Src)
		set.facilityScenes[resource] = &modernFacilityScene{image: scene}
		set.portraits[speaker] = &modernStoryPortraitFrame{image: image.NewRGBA(image.Rect(0, 0, 80, 80)), hdImage: head}
	}
	if err := validateModernFacilityPortraitScenePairs(set); err != nil {
		t.Fatal(err)
	}
	for _, owner := range []string{"hotel", "church"} {
		g := &Game{nativeClassUI: shared, nativeHotelUI: hotel, modernStoryPortraits: set, nativeHotelMode: "menu", churchMode: "menu", nativeChurchTextIndex: 586,
			camp: campaign.NewRunner(&campaign.Campaign{Start: owner, Nodes: map[string]*campaign.Node{owner: {Type: owner}}})}
		compose := g.composeNativeChurchMenuFrame
		speaker := 131
		if owner == "hotel" {
			compose = func() ([]byte, bool) { return g.composeNativeHotelFrame(0) }
			speaker = 129
		}
		before, ok := compose()
		if !ok {
			t.Fatal("native source missing", owner)
		}
		source := g.modernStableFacilitySource()
		if source == nil {
			t.Fatal("proved pair unavailable", owner)
		}
		g.nativeShopUIJob = &nativeClassUIJob{}
		if g.modernStableFacilitySource() != nil {
			t.Fatal("foreign active job admitted", owner)
		}
		g.nativeShopUIJob = nil
		classPointer, hotelPointer := g.nativeClassUI, g.nativeHotelUI
		a, okA := g.composeModernFacilityMarker(source, 0, compose)
		b, okB := g.composeModernFacilityMarker(source, 255, compose)
		if !okA || !okB || g.nativeClassUI != classPointer || g.nativeHotelUI != hotelPointer {
			t.Fatal("marker source pointer leaked", owner)
		}
		offset := campaign.NativeFacilityPortraitOffset(speaker)
		for y := 0; y < 80; y++ {
			for x := 0; x < 80; x++ {
				i := offset + y*320 + x
				if a[i] == b[i] {
					t.Fatal("native face masked HD full scene", owner, x, y)
				}
			}
		}
		for y := 112; y < 198; y++ {
			for x := 5; x < 315; x++ {
				if a[y*320+x] != b[y*320+x] {
					t.Fatal("native dialogue writer lost", owner, x, y)
				}
			}
		}
		if a[0] == b[0] {
			t.Fatal("untouched background became foreground", owner)
		}
		// A later UI write remains opaque even if it equals the zero marker.
		covered := func() ([]byte, bool) {
			f, ok := compose()
			if ok {
				f[offset+10*320+10] = 0
			}
			return f, ok
		}
		c, _ := g.composeModernFacilityMarker(source, 0, covered)
		d, _ := g.composeModernFacilityMarker(source, 255, covered)
		if c[offset+10*320+10] != 0 || d[offset+10*320+10] != 0 {
			t.Fatal("later foreground write was removed", owner)
		}
		after, ok := compose()
		if !ok || !bytes.Equal(before, after) {
			t.Fatal("marker passes changed original frame", owner)
		}
		if owner == "hotel" {
			copyUI := *hotel
			copyAssets := *hotel.assets
			copyAssets.Background = append([]byte(nil), hotel.assets.Background...)
			copyAssets.Background[0] ^= 1
			copyUI.assets = &copyAssets
			g.nativeHotelUI = &copyUI
		} else {
			copyUI := *shared
			copyUI.background = append([]byte(nil), shared.background...)
			copyUI.background[0] ^= 1
			g.nativeClassUI = &copyUI
		}
		if g.modernStableFacilitySource() != nil {
			t.Fatal("changed original source reused HD scene", owner)
		}
	}
	for _, fault := range []string{"missing-scene", "missing-head", "shape", "crop"} {
		scene := *set.facilityScenes[13]
		portrait := *set.portraits[129]
		bad := &modernStoryPortraitSet{facilityScenes: map[int]*modernFacilityScene{13: &scene}, portraits: map[int]*modernStoryPortraitFrame{129: &portrait}}
		switch fault {
		case "missing-scene":
			delete(bad.facilityScenes, 13)
		case "missing-head":
			delete(bad.portraits, 129)
		case "shape":
			portrait.hdImage = image.NewRGBA(image.Rect(0, 0, 80, 80))
		case "crop":
			head := image.NewRGBA(image.Rect(0, 0, 480, 480))
			draw.Draw(head, head.Bounds(), portrait.hdImage, image.Point{}, draw.Src)
			head.SetRGBA(479, 479, color.RGBA{R: 255, A: 255})
			portrait.hdImage = head
		}
		if validateModernFacilityPortraitScenePairs(bad) == nil {
			t.Fatal("invalid pair published", fault)
		}
	}
	if err := validateModernFacilityPortraitScenePairs(&modernStoryPortraitSet{}); err != nil {
		t.Fatal("old catalog unexpectedly needs new art", err)
	}
}

func TestModernHDFacilitySceneCatalogPublication(t *testing.T) {
	path, pack := os.Getenv("FD2_TEST_FACILITY_CATALOG"), os.Getenv("FD2_TEST_FACILITY_PACK")
	if path == "" || pack == "" {
		t.Skip("explicit facility catalog required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	assets := catalog.Assets
	catalog.Assets = nil
	for _, asset := range assets {
		if asset.Role == "native_ui_icon_hd" || asset.Role == "facility_scene_hd" ||
			((asset.Role == "story_portrait_frame" || asset.Role == "story_portrait_hd_frame") && (asset.SpeakerID == 0 || asset.SpeakerID == 129 || asset.SpeakerID == 131)) {
			catalog.Assets = append(catalog.Assets, asset)
		}
	}
	minimal := filepath.Join(t.TempDir(), "catalog.json")
	write := func() {
		raw, err := json.Marshal(catalog)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(minimal, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	set, err := loadModernStoryPortraitSet(minimal, pack)
	if err != nil || len(set.facilityScenes) != 8 {
		t.Fatal("new paired scene catalog not published", err)
	}
	kept := catalog.Assets
	catalog.Assets = nil
	for _, asset := range kept {
		if (asset.Role == "facility_scene_hd" && (asset.SourceResource == 13 || asset.SourceResource == 14)) ||
			((asset.Role == "story_portrait_frame" || asset.Role == "story_portrait_hd_frame") && (asset.SpeakerID == 129 || asset.SpeakerID == 131)) {
			continue
		}
		catalog.Assets = append(catalog.Assets, asset)
	}
	write()
	set, err = loadModernStoryPortraitSet(minimal, pack)
	if err != nil || len(set.facilityScenes) != 6 {
		t.Fatal("legacy six scenes no longer load", err)
	}
}
