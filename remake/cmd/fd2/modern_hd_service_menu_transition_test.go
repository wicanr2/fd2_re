package main

import (
	"bytes"
	"image/color"
	"testing"
)

func serviceMenuTestDescriptor(g *Game, owner string) *nativeServiceMenuPresentation {
	if owner == "shop" {
		if g.nativeShopUIJob != nil {
			return g.nativeShopUIJob.serviceMenu
		}
	} else if g.nativeChurchUIJob != nil {
		return g.nativeChurchUIJob.serviceMenu
	}
	return nil
}
func serviceMenuBoundTestJob(t *testing.T, g *Game, owner string, variant int, closing bool) *nativeServiceMenuPresentation {
	t.Helper()
	serviceMenuTestState(g, owner, variant)
	if !serviceMenuTestBegin(g, owner, closing, func() {}) {
		t.Fatal("native menu begin")
	}
	d := serviceMenuTestDescriptor(g, owner)
	if d == nil {
		t.Fatal("menu did not bind", owner, variant, closing)
	}
	return d
}
func TestModernHDServiceMenuSourceAndFallback(t *testing.T) {
	set := minimalServiceIconSet(t)
	g := facilityDialogueTestGame(t, nil)
	g.modernThemeCache = set
	g.interfaceMode = playerInterfaceHD
	cases, frames, restores, coverage, rejected := 0, 0, 0, 0, 0
	for _, owner := range []string{"shop", "church"} {
		variants := []int{1, 3, 5}
		if owner == "church" {
			variants = []int{0}
		}
		for _, variant := range variants {
			for _, closing := range []bool{false, true} {
				g.modernHDText = nil
				d := serviceMenuBoundTestJob(t, g, owner, variant, closing)
				actual, restore := serviceMenuTestCache(g, owner)
				for i, frame := range actual {
					p, ok := d.compose(i, false)
					if !ok || !bytes.Equal(p.pixels, frame) {
						t.Fatal("native menu cache changed")
					}
					p, ok = d.compose(i, true)
					if !ok || len(p.runs) == 0 || len(p.panels) != 1 {
						t.Fatal("HD menu source absent")
					}
					icons, ok := g.modernServiceMenuIcons(d, i)
					if !ok || len(icons) != 4 {
						t.Fatal("HD menu normal sources absent")
					}
					for option, placement := range icons {
						c := d.cells[option]
						picture := placement.icon.picture
						for y := 0; y < 120; y++ {
							for x := 0; x < 144; x++ {
								alpha := byte(255)
								if c.Pixels[(y/6)*24+x/6] == 0 {
									alpha = 0
								}
								if picture.Pix[picture.PixOffset(x, y)+3] != alpha {
									t.Fatal("HD mask differs from native index zero")
								}
								coverage++
							}
						}
					}
					frames++
				}
				p, ok := d.compose(4, false)
				if closing {
					if !ok || !bytes.Equal(p.pixels, restore) {
						t.Fatal("source restore changed")
					}
					restores++
				} else if ok {
					t.Fatal("invented restore")
				}
				d.lastDraw = &modernChurchConfirmationDraw{frame: 99}
				if g.drawModernHDFacilityServiceMenu(nil, d) || d.lastDraw != nil {
					t.Fatal("original kept HD Draw")
				}
				for _, fault := range []string{"mode", "owner", "job", "foreign-job", "locale", "font", "palette", "raw", "restore", "help", "transient", "black", "white", "chrome", "icon", "icon-pixels", "coverage"} {
					if fault == "restore" && !closing {
						continue
					}
					font, palette, chrome := g.font, g.nativeClassUI.palette, d.chrome
					icon := set.uiServiceIcons[[2]int{d.resource, 3}]
					g.modernHDText = &modernHDTextFrame{}
					switch fault {
					case "mode":
						if owner == "shop" {
							g.nativeShopMode = "unbound"
						} else {
							g.churchMode = "unbound"
						}
					case "owner":
						g.camp.Node().Type = "hotel"
					case "job":
						if owner == "shop" {
							g.nativeShopUIJob = &nativeClassUIJob{}
						} else {
							g.nativeChurchUIJob = &nativeChurchUIJob{}
						}
					case "foreign-job":
						g.nativeClassUIJob = &nativeClassUIJob{}
					case "locale":
						g.localeID = "unbound"
					case "font":
						g.font = nil
					case "palette":
						g.nativeClassUI.palette = append(color.Palette(nil), palette...)
						g.nativeClassUI.palette[73] = nil
					case "raw":
						actual[0][0] ^= 1
					case "restore":
						restore[0] ^= 1
						if owner == "shop" {
							d.shopJob.frame = 4
						} else {
							d.churchJob.frame = 4
						}
					case "chrome":
						d.chrome = nil
					case "icon":
						delete(set.uiServiceIcons, [2]int{d.resource, 3})
					case "icon-pixels":
						icon.picture.Pix[0] ^= 1
					case "coverage":
						d.icons[0].picture.Pix[0] ^= 1
					}
					g.helpVisible, g.transientUI = fault == "help", fault == "transient"
					g.nativeFullDACBlack, g.nativeFullDACWhite = fault == "black", fault == "white"
					d.lastDraw = &modernChurchConfirmationDraw{frame: 99}
					if g.drawModernHDFacilityServiceMenu(nil, d) || d.lastDraw != nil {
						t.Fatal("invalid source retained HD Draw", owner, fault)
					}
					if fault == "raw" {
						actual[0][0] ^= 1
					}
					if fault == "restore" {
						restore[0] ^= 1
					}
					if fault == "icon-pixels" {
						icon.picture.Pix[0] ^= 1
					}
					if fault == "coverage" {
						d.icons[0].picture.Pix[0] ^= 1
					}
					set.uiServiceIcons[[2]int{d.resource, 3}] = icon
					g.font, g.nativeClassUI.palette, d.chrome = font, palette, chrome
					g.localeID = d.locale
					g.camp.Node().Type = owner
					g.nativeShopMode, g.churchMode = "menu", "menu"
					g.nativeClassUIJob = nil
					if owner == "shop" {
						g.nativeShopUIJob, g.nativeChurchUIJob = d.shopJob, nil
						d.shopJob.frame = 0
					} else {
						g.nativeChurchUIJob, g.nativeShopUIJob = d.churchJob, nil
						d.churchJob.frame = 0
					}
					rejected++
				}
				g.modernHDText = nil
				g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
				g.nativeShopUIJob, g.nativeChurchUIJob = nil, nil
				cases++
			}
		}
	}
	t.Logf("menu cases=%d native_frames=%d restores=%d hd_coverage_pixels=%d fault_guards=%d", cases, frames, restores, coverage, rejected)
}
