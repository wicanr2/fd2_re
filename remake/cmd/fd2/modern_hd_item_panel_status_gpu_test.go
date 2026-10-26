package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type itemPanelStatusGPUProbe struct {
	set                  *modernStoryPortraitSet
	g, control           *Game
	final, controlScreen *ebiten.Image
	pixels               []byte
	colors               dialogueChromeSource
	out                  string
	rows                 []map[string]any
	seen                 map[string]bool
	f2, points           int
	valuePoints          int
	banks                []*nativeUnitStatusData
	bankRows             []map[string]any
	bankIndex            int
	done                 bool
	err                  error
}

func (p *itemPanelStatusGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *itemPanelStatusGPUProbe) Update() error {
	if p.done && p.bankIndex == len(p.banks) || p.err != nil {
		return ebiten.Termination
	}
	return nil
}

func itemPanelStatusGPUState(g *Game) string {
	var values []any
	if s := g.nativePlayerStatus; s != nil {
		values = []any{rewardStageNativeState(g), s.phase, s.frame, s.drawn, len(s.frames), sha256.Sum256(s.source), sha256.Sum256(s.status), sha256.Sum256(s.commands)}
	} else {
		values = []any{rewardStageNativeState(g), "return"}
	}
	raw, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func (p *itemPanelStatusGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil {
		return
	}
	if p.done {
		if p.bankIndex < len(p.banks) {
			p.drawRowLabelBank(screen)
		}
		return
	}
	g, c := p.g, p.control
	g.frame++
	c.frame++
	g.Draw(screen)
	c.Draw(p.controlScreen)
	if g.loadErr != "" || c.loadErr != "" || itemPanelStatusGPUState(g) != itemPanelStatusGPUState(c) {
		p.err = fmt.Errorf("status native controller differs: errors=%q/%q work=%x/%x vga=%x/%x", g.loadErr, c.loadErr, sha256.Sum256(g.nativeMapWork), sha256.Sum256(c.nativeMapWork), sha256.Sum256(g.nativeMapVGA), sha256.Sum256(c.nativeMapVGA))
		return
	}
	phase, index := "return", -1
	var regions []battle.NativeItemPanelRegion
	s := g.nativePlayerStatus
	if s != nil {
		phase, index = s.phase, s.frame
		var ok bool
		if len(s.frames) != 0 {
			regions, _, ok = nativeStatusTransitionPass(s.text.kind, s.frame, s.text.commands)
		} else {
			regions, ok = nativeChurchPanelRegions(0)
		}
		if !ok {
			p.err = fmt.Errorf("status native pass unavailable")
			return
		}
	}
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	presentation := &modernHDPresentation{Game: g}
	p.final.Clear()
	presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(p.pixels)
	points, err := checkItemPanelRegionChromePixels(p.pixels, p.colors, regions)
	if err != nil || s != nil && points == 0 {
		p.err = fmt.Errorf("status %s/%d chrome: %v", phase, index, err)
		return
	}
	p.points += points
	valuePoints := 0
	var valuesData *nativeUnitStatusData
	if s != nil {
		valuesData = s.data
		if len(s.frames) != 0 {
			valuesData = &s.text.data
		}
		valuePoints, err = checkItemPanelValuesPixels(p.pixels, valuesData, g.nativeUIPalette, regions)
		if err != nil {
			p.err = err
			return
		}
		p.valuePoints += valuePoints
	}
	key := phase
	if s != nil && len(s.frames) != 0 {
		if s.frame == len(s.frames)-1 {
			key += "-last"
		}
	}
	f2 := !p.seen[key]
	digest := sha256.Sum256(p.pixels)
	if f2 {
		state := itemPanelStatusGPUState(g)
		if err := g.applyTheme(nil); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDText != nil || g.modernHDOverlay != nil {
			p.err = fmt.Errorf("status HD survived pixel switch")
			return
		}
		if err := g.applyTheme(p.set); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		p.final.Clear()
		presentation.DrawFinalScreen(p.final, screen, geometry)
		p.final.ReadPixels(p.pixels)
		if sha256.Sum256(p.pixels) != digest || state != itemPanelStatusGPUState(g) {
			p.err = fmt.Errorf("status F2 changed pixels or state")
			return
		}
		p.seen[key] = true
		p.f2++
	}
	row := map[string]any{"phase": phase, "frame": index, "regions": regions, "chrome_pixels": points, "f2_roundtrip": f2, "final_sha256": fmt.Sprintf("%x", digest), "native_control_unchanged": true}
	row["item_values_pixels"] = valuePoints
	if valuesData != nil {
		row["item_values_record"] = fmt.Sprintf("%x", valuesData.record)
		row["item_values_colors"] = itemPanelValuesColors(valuesData, g.nativeUIPalette)
	}
	commands := s != nil && s.phase == nativePlayerStatusCommands
	if s != nil && len(s.frames) != 0 {
		_, commands, _ = nativeStatusTransitionPass(s.text.kind, s.frame, s.text.commands)
	}
	if err := addRowLabelFrameMetadata(row, valuesData, commands); err != nil {
		p.err = err
		return
	}
	if f2 {
		name := "status-" + key + ".png"
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, p.pixels)
		f, err := os.Create(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		err = png.Encode(f, pic)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			p.err = fmt.Errorf("status PNG: %v %v", err, closeErr)
			return
		}
		row["capture"] = name
	}
	p.rows = append(p.rows, row)
	if len(p.rows) > 80 {
		p.err = fmt.Errorf("unbounded status lifecycle")
		return
	}
	if s == nil {
		p.done = true
		return
	}
	for _, game := range []*Game{g, c} {
		if len(game.nativePlayerStatus.frames) != 0 {
			game.stepNativePlayerStatus()
		} else {
			game.advanceNativePlayerStatus()
		}
	}
}

func TestModernHDItemPanelStatusGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_ITEM_PANEL_STATUS_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	set := itemHDSet(t)
	g, _ := rewardStageGame(t, set)
	c, _ := rewardStageGame(t, nil)
	for _, game := range []*Game{g, c} {
		u := makeNativeDeathRewardKiller(t, game)
		// The shared battlefield fixture owns command 9 in State, while the
		// status UI reads Game's command list. Supply that same existing row.
		game.nativeCommandBook = make([]battle.NativeCommandRecord, 10)
		game.nativeCommandBook[9] = game.st.NativeCommandBook[9]
		u.NativeCommandMask = [5]byte{0, 2}
		if game.nativeCommandBook[9].ID != 9 {
			t.Fatal("status fixture command 9 is unavailable")
		}
		if err := game.beginNativePlayerStatus(u); err != nil {
			t.Fatal(err)
		}
	}
	out := os.Getenv("FD2_TEST_REWARD_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &itemPanelStatusGPUProbe{set: set, g: g, control: c, final: ebiten.NewImage(1920, 1200), controlScreen: ebiten.NewImage(640, 400), pixels: make([]byte, 1920*1200*4), out: out, seen: map[string]bool{}, colors: nativeDialogueChromeSource(g.nativePreparationUI.dialogue, g.nativeUIPalette)}
	if os.Getenv("FD2_HD_ROW_LABEL_GPU") == "1" {
		for i := 0; i < 3; i++ {
			data := mixedHDRowLabelData(t, g.nativePlayerStatus.data)
			if os.Getenv("FD2_HD_PANEL_ICON_GPU") == "1" {
				data.record[6] = byte(i % 2)
				data.record[37], data.record[38], data.record[39] = 1, 1, 1
			}
			if i == 2 {
				for slot := 0; slot < 8; slot++ {
					data.record[10+2*slot] = 0x80
				}
			}
			p.banks = append(p.banks, data)
		}
	}
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || !p.done || p.f2 != 9 || len(p.rows) != 41 || p.points <= 0 || p.valuePoints <= 0 {
		t.Fatal("incomplete status lifecycle", p.err, p.f2, len(p.rows), p.points)
	}
	if p.bankIndex != len(p.banks) || len(p.bankRows) != len(p.banks) {
		t.Fatal("incomplete row label banks", p.bankIndex, len(p.banks))
	}
	report := map[string]any{"status": "PASS-ITEM-PANEL-STATUS-HD-RUNTIME-E1", "rows": p.rows, "frames": len(p.rows), "f2_roundtrips": p.f2, "chrome_pixels": p.points, "chrome_source": p.colors, "normal_player_verified": false, "native_seed": 7, "fixed_clock_unix": 1000, "command_mask": []int{0, 2, 0, 0, 0}, "command_book_source": "State.NativeCommandBook[9] from existing player9StageGame fixture", "map_index": 23}
	report["item_values"], report["item_values_pixels"] = true, p.valuePoints
	report["row_labels"], report["row_label_banks"] = true, p.bankRows
	report["panel_icons"] = len(set.uiPanelIcons) == 10
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
