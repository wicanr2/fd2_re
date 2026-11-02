package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func minimalServiceIconSet(t *testing.T) *modernStoryPortraitSet {
	t.Helper()
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_FACILITY_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	var entries []modernThemeAsset
	for _, a := range catalog.Assets {
		if a.Role == "native_ui_icon_hd" || (a.Role == "story_portrait_frame" || a.Role == "story_portrait_hd_frame") && a.SpeakerID == 0 {
			entries = append(entries, a)
		}
	}
	catalog.Assets = entries
	path := filepath.Join(t.TempDir(), "catalog.json")
	data, _ := json.Marshal(catalog)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	set, err := loadModernStoryPortraitSet(path, os.Getenv("FD2_TEST_FACILITY_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	if len(set.uiPanelIcons) != 10 || len(set.uiServiceIcons) != 40 {
		t.Fatal("new source groups not independently published")
	}
	// A legacy catalog has no optional service group and keeps all old panels.
	var legacy []modernThemeAsset
	for _, a := range catalog.Assets {
		if a.Role != "native_ui_icon_hd" || a.SourceResource == 5 {
			legacy = append(legacy, a)
		}
	}
	catalog.Assets = legacy
	data, _ = json.Marshal(catalog)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	old, err := loadModernStoryPortraitSet(path, os.Getenv("FD2_TEST_FACILITY_PACK"))
	if err != nil || len(old.uiPanelIcons) != 10 || len(old.uiServiceIcons) != 0 {
		t.Fatal("legacy publication failed", err)
	}
	return set
}

func TestModernHDServiceIconsSourceAndPublication(t *testing.T) {
	set := minimalServiceIconSet(t)
	g := facilityDialogueTestGame(t, set)
	g.modernThemeCache = set
	for _, resource := range []int{12, 13, 14, 29, 63} {
		owner := "shop"
		mode := "menu"
		if resource == 13 {
			owner = "hotel"
		}
		if resource == 14 {
			owner = "church"
		}
		for selection := 0; selection < 4; selection++ {
			for phase := 0; phase < 4; phase++ {
				facilityDialogueTestOwner(g, owner, mode, selection, phase)
				if owner == "shop" {
					g.nativeShopMode = "menu"
					g.nativeShopVariant = map[int]int{12: 1, 29: 3, 63: 5}[resource]
					g.nativeShopServiceSel = selection
				}
				frame, ok := facilityDialogueCompose(g)
				if owner == "shop" {
					frame, ok = g.composeNativeShopServiceMenu()
				}
				if !ok {
					t.Fatal("native menu absent", resource)
				}
				before := append([]byte(nil), frame...)
				g.interfaceMode = playerInterfaceHD
				g.modernHDText = &modernHDTextFrame{}
				g.appendModernHDServiceIcons(frame)
				if len(g.modernHDText.serviceIcons) != 4 {
					t.Fatal("source not published", resource, selection, phase)
				}
				for option, p := range g.modernHDText.serviceIcons {
					pulse := 0
					if option == selection {
						pulse = phase / 2
					}
					if p.icon != set.uiServiceIcons[[2]int{resource, 3 + 2*option + pulse}] || p.origin != image.Pt(201+26*option, 169) {
						t.Fatal("wrong raw source/placement")
					}
				}
				covered := append([]byte(nil), frame...)
				covered[169*320+201] ^= 1
				g.appendModernHDServiceIcons(covered)
				if len(g.modernHDText.serviceIcons) != 0 {
					t.Fatal("covered menu kept icons")
				}
				if !bytes.Equal(before, frame) {
					t.Fatal("source frame changed")
				}
				g.nativeShopUIJob = &nativeClassUIJob{}
				g.appendModernHDServiceIcons(frame)
				if len(g.modernHDText.serviceIcons) != 0 {
					t.Fatal("foreign job admitted")
				}
				g.nativeShopUIJob = nil
				g.interfaceMode = playerInterfaceOriginal
				g.modernHDText = nil
				g.appendModernHDServiceIcons(frame)
				if g.modernHDText != nil {
					t.Fatal("original UI created HD layer")
				}
			}
		}
	}
	facilityDialogueTestOwner(g, "church", "menu", 0, 0)
	frame, ok := g.composeNativeChurchMenuFrame()
	if !ok {
		t.Fatal("menu")
	}
	g.interfaceMode = playerInterfaceHD
	g.modernHDText = &modernHDTextFrame{}
	original := g.nativeClassUI
	copyUI := *original
	copyUI.palette = append(color.Palette(nil), original.palette...)
	copyUI.palette[74] = color.RGBA{40, 69, 137, 255}
	g.nativeClassUI = &copyUI
	g.appendModernHDServiceIcons(frame)
	if len(g.modernHDText.serviceIcons) != 0 {
		t.Fatal("different menu background palette admitted")
	}
	g.nativeClassUI = original
	icon := set.uiServiceIcons[[2]int{14, 3}]
	pixel := icon.picture.Pix[0]
	icon.picture.Pix[0] ^= 1
	g.appendModernHDServiceIcons(frame)
	if len(g.modernHDText.serviceIcons) != 0 {
		t.Fatal("changed cached PNG admitted")
	}
	icon.picture.Pix[0] = pixel
	delete(set.uiServiceIcons, [2]int{14, 3})
	if validateModernHDServiceIconGroup(set) == nil {
		t.Fatal("incomplete group admitted")
	}
	g.appendModernHDServiceIcons(frame)
	if len(g.modernHDText.serviceIcons) != 0 {
		t.Fatal("missing optional source kept partial menu")
	}
	set.uiServiceIcons[[2]int{14, 3}] = icon
	if icon.native != modernUIRawDescription(3, fdother.RawCell{Width: 24, Height: 20, Pixels: g.nativeClassUI.entries[3].Pixels}) {
		t.Fatal("actual church raw source differs")
	}
}

func TestModernHDServiceIconsRejectCorruptAssets(t *testing.T) {
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
		if a.Role == "native_ui_icon_hd" && a.SourceResource == 14 && a.NativeRawCell.Index == 3 {
			source = a
			break
		}
	}
	if source.File == "" {
		t.Fatal("service source absent")
	}
	for _, fault := range []string{"resource", "metadata", "raw-pixels", "geometry", "hash", "path", "contract", "duplicate"} {
		a := source
		target := &modernStoryPortraitSet{}
		switch fault {
		case "resource":
			a.SourceResource = 42
		case "metadata":
			a.SourceMetadataSHA256 = "bad"
		case "raw-pixels":
			a.NativeRawCell.PixelsSHA256 = "bad"
		case "geometry":
			a.Width++
		case "hash":
			a.SHA256 = "bad"
		case "path":
			a.File = "../" + a.File
		case "contract":
			a.ConsumerContract = "unproved"
		case "duplicate":
			target.uiServiceIcons = map[[2]int]*modernHDServiceIcon{{14, 3}: {}}
		}
		if err := loadModernUIPanelIcon(target, a, os.Getenv("FD2_TEST_FACILITY_PACK")); err == nil {
			t.Fatal("accepted", fault)
		}
	}
}

func TestModernHDServiceIconsRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SERVICE_ICONS_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	set := facilityHDTestSet(t)
	legacy := *set
	legacy.uiServiceIcons = nil
	g := facilityDialogueTestGame(t, set)
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true},
		captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, _ *ebiten.Image) error {
		resource, selected, phase, cells, ok := g.modernServiceMenuSource()
		if !ok {
			return fmt.Errorf("case%d actual service source absent", p.index)
		}
		hd := g.interfaceMode == playerInterfaceHD && len(g.modernThemeCache.uiServiceIcons) == 40
		count := 0
		if g.modernHDText != nil {
			count = len(g.modernHDText.serviceIcons)
		}
		if (count == 4) != hd || !hd && count != 0 {
			return fmt.Errorf("case%d icon publication %d, HD=%v", p.index, count, hd)
		}
		checked := 0
		for option := 0; option < 4; option++ {
			pulse := 0
			if option == selected {
				pulse = phase / 2
			}
			index := 3 + 2*option + pulse
			for y := 0; y < 120; y++ {
				for x := 0; x < 144; x++ {
					var want color.RGBA
					if hd {
						want = set.uiServiceIcons[[2]int{resource, index}].picture.RGBAAt(x, y)
					} else {
						value := cells[option][pulse].Pixels[(y/6)*24+x/6]
						if value == 0 {
							value = 74
						}
						want = color.RGBAModel.Convert(g.nativeClassUI.palette[value]).(color.RGBA)
					}
					at := (((169*6+y)+72)*1920 + (201+26*option)*6 + x) * 4
					if !bytes.Equal(pixels[at:at+4], []byte{want.R, want.G, want.B, want.A}) {
						return fmt.Errorf("case%d resource%d entry%d pixel%d,%d: %v != %v", p.index, resource, index, x, y, pixels[at:at+4], want)
					}
					checked++
				}
			}
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "resource": resource, "selection": selected,
			"phase": phase, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil,
			"service_group": len(g.modernThemeCache.uiServiceIcons), "hd_icons": hd, "icon_pixels": checked})
		return nil
	}
	add := func(resource, selection, phase, style int, old bool) {
		if style == 3 && selection == 0 && !old {
			p.captureCases[len(p.cases)] = true
		}
		p.cases = append(p.cases, func() {
			owner := "shop"
			if resource == 13 {
				owner = "hotel"
			} else if resource == 14 {
				owner = "church"
			}
			facilityDialogueTestOwner(g, owner, "menu", selection, phase)
			if owner == "shop" {
				g.nativeShopMode, g.nativeShopServiceSel = "menu", selection
				g.nativeShopVariant = map[int]int{12: 1, 29: 3, 63: 5}[resource]
			}
			active := set
			if old {
				active = &legacy
			}
			g.modernThemeCache, g.modernStoryPortraits = active, nil
			if style >= 2 {
				g.modernStoryPortraits = active
			}
			g.interfaceMode = playerInterfaceOriginal
			if style%2 == 1 {
				g.interfaceMode = playerInterfaceHD
			}
		})
	}
	for _, resource := range []int{12, 13, 14, 29, 63} {
		for selection := 0; selection < 4; selection++ {
			for _, phase := range []int{0, 2} {
				for style := 0; style < 4; style++ {
					add(resource, selection, phase, style, false)
				}
			}
		}
		for _, style := range []int{1, 3} {
			add(resource, 0, 2, style, true)
		}
	}
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != 170 {
		t.Fatal("incomplete source/style/fallback matrix", len(p.checks))
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-SERVICE-ICONS-GPU",
		"scope": "actual Game.Draw and final presentation; five resources, four selections, two pulses, four independent styles and ten legacy-group fallbacks; toolbar viewport unchanged; no new original oracle claim", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
