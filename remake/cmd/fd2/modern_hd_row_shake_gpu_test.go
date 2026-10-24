package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type rowShakeGPUCase struct {
	name                        string
	count                       int
	g, control                  *Game
	continued, controlContinued *int
	draws, switches             int
	want                        [2]*image.RGBA
	returnSHA                   string
}

type rowShakeGPUProbe struct {
	checker       unitPresentGPUProbe
	cases         []rowShakeGPUCase
	index         int
	out           string
	controlScreen *ebiten.Image
	rows          []map[string]any
}

func (p *rowShakeGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *rowShakeGPUProbe) Update() error {
	if p.checker.err != nil || p.index >= len(p.cases) {
		return ebiten.Termination
	}
	return nil
}

func (p *rowShakeGPUProbe) save(name string, pixels []byte) error {
	f, err := os.Create(filepath.Join(p.out, name))
	if err != nil {
		return err
	}
	picture := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	copy(picture.Pix, pixels)
	err = png.Encode(f, picture)
	closed := f.Close()
	if err == nil {
		err = closed
	}
	return err
}

func (p *rowShakeGPUProbe) Draw(screen *ebiten.Image) {
	if p.checker.err != nil || p.index >= len(p.cases) {
		return
	}
	c := &p.cases[p.index]
	g, j := c.g, c.g.transitionReveal
	if j == nil {
		p.checker.err = fmt.Errorf("lost row owner")
		return
	}
	row, step := j.index, c.draws
	before := rowShakeNativeState(g)
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || !j.drawn || !reflect.DeepEqual(before, rowShakeNativeState(g)) {
		p.checker.err = fmt.Errorf("%s draw changed source: %s", c.name, g.loadErr)
		return
	}
	frame := j.hdFrames[row]
	pixels, ok := p.checker.checkWorld(g, frame, c.want[row], screen, c.name)
	if !ok {
		return
	}
	digest := sha256.Sum256(pixels)
	if step == 0 || step == 1 || step == c.count-1 {
		if err := g.applyTheme(nil); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.checker.err = fmt.Errorf("native theme retained row HD")
			return
		}
		if err := g.applyTheme(p.checker.set); err != nil {
			p.checker.err = err
			return
		}
		g.Draw(screen)
		again, ok := p.checker.checkWorld(g, g.transitionReveal.hdFrames[row], c.want[row], screen, c.name+" F2")
		if !ok {
			return
		}
		if digest != sha256.Sum256(again) || !reflect.DeepEqual(before, rowShakeNativeState(g)) {
			p.checker.err = fmt.Errorf("F2 changed row, state or pixels")
			return
		}
		c.switches++
	}
	if p.controlScreen == nil {
		p.controlScreen = ebiten.NewImage(640, 400)
	}
	p.controlScreen.Clear()
	c.control.Draw(p.controlScreen)
	if !c.control.transitionReveal.drawn || !reflect.DeepEqual(rowShakeNativeState(g), rowShakeNativeState(c.control)) {
		p.checker.err = fmt.Errorf("native control diverged")
		return
	}
	p.rows = append(p.rows, map[string]any{"case": c.name, "step": step, "row": row, "final_sha256": fmt.Sprintf("%x", digest), "native_vga_sha256": fmt.Sprintf("%x", sha256.Sum256(g.nativeMapVGA)), "rng": g.nativeRNGState})
	if step < 2 {
		if err := p.save(fmt.Sprintf("%s-row%d.png", c.name, row), pixels); err != nil {
			p.checker.err = err
			return
		}
	}
	c.draws++
	for _, game := range []*Game{g, c.control} {
		game.stepTransitionReveal()
		game.frame++
	}
	if g.transitionReveal == nil {
		if c.control.transitionReveal != nil || *c.continued != 1 || *c.controlContinued != 1 || c.draws != c.count || c.switches != 3 {
			p.checker.err = fmt.Errorf("wrong completion boundary")
			return
		}
		g.Draw(screen)
		if !g.modernHDWorldVisible || g.modernHDMapPresented == nil {
			p.checker.err = fmt.Errorf("HD return absent: %s", g.loadErr)
			return
		}
		pixels, ok := p.checker.checkWorld(g, g.modernHDMapPresented, healHDCPUWorld(g.modernHDMapPresented, p.checker.set), screen, c.name+" return")
		if !ok {
			return
		}
		c.returnSHA = fmt.Sprintf("%x", sha256.Sum256(pixels))
		if err := p.save(c.name+"-return.png", pixels); err != nil {
			p.checker.err = err
			return
		}
		p.index++
	}
}

func TestModernHDRowShakeGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_ROW_SHAKE_GPU") != "1" {
		t.Skip("dedicated row-shake GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	out := os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_UNIT_PRESENT_CATALOG"), os.Getenv("FD2_TEST_UNIT_PRESENT_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	p := &rowShakeGPUProbe{out: out, checker: unitPresentGPUProbe{set: set}}
	for _, story := range []bool{false, true} {
		for _, count := range []int{20, 30} {
			name := fmt.Sprintf("battle-%d", count)
			if story {
				name = fmt.Sprintf("story-%d", count)
			}
			games := make([]*Game, 2)
			for i := range games {
				if story {
					games[i] = indexedHDActualGame(t, nil)
				} else {
					games[i], _, _ = ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
					games[i].nativeRNGState = 7
					games[i].st.MaterializeNativeMapRangeMode(1)
					v := games[i].st.NativeMapViewState
					u := games[i].st.Units[0]
					v.CursorX, v.CursorY = u.X, u.Y
					v.VisibleCursorX, v.VisibleCursorY = u.X-v.CameraX, u.Y-v.CameraY
					games[i].curX, games[i].curY = u.X, u.Y
					if err := games[i].st.MaterializeNativeMapViewState(v); err != nil {
						t.Fatal(err)
					}
					games[i].st.MaterializeNativeMapHUDState(1, 1, 1)
				}
			}
			g, control := games[0], games[1]
			g.modernStoryPortraits = set
			a, b := new(int), new(int)
			if err := g.startNative24B4D(count, 20, func() { *a++ }); err != nil {
				t.Fatal(name, err)
			}
			if err := control.startNative24B4D(count, 20, func() { *b++ }); err != nil {
				t.Fatal(name, err)
			}
			j := g.transitionReveal
			c := rowShakeGPUCase{name: name, count: count, g: g, control: control, continued: a, controlContinued: b}
			for row := 0; row < 2; row++ {
				c.want[row] = healHDCPUWorld(j.hdFrames[row], set)
			}
			// Independent projection: every overlapping HD pixel must move six rows.
			for y := 24; y < 1170; y++ {
				for x := 24; x < 1896; x++ {
					if c.want[1].RGBAAt(x, y) != c.want[0].RGBAAt(x, y+6) {
						t.Fatal("whole world row projection differs", name, x, y)
					}
				}
			}
			extra := false
			for _, tile := range j.hdFrames[1].Terrain {
				if tile.WorkOffset >= 0x8088+192*456 && tile.Rect.Min.Y == 1170 {
					extra = true
				}
			}
			if !extra || len(j.hdFrames[0].Units) == 0 || !story && (j.hdFrames[0].HUDUnit == nil || j.hdFrames[0].Range == nil) {
				t.Fatal("missing real ninth-row HD, unit, HUD or range", name)
			}
			p.cases = append(p.cases, c)
		}
	}
	ebiten.SetWindowSize(1920, 1200)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.checker.err != nil {
		t.Fatal(p.checker.err)
	}
	cases := make([]map[string]any, len(p.cases))
	for i, c := range p.cases {
		cases[i] = map[string]any{"name": c.name, "draws": c.draws, "f2_round_trips": c.switches, "continuations": *c.continued, "native_control_unchanged": true, "return_final_sha256": c.returnSHA}
	}
	raw, err := json.MarshalIndent(map[string]any{"status": "PASS-ROW-SHAKE-HD-GAME", "fixed_clock_unix": 1000, "native_seed": 7, "rgba_tolerance": 2, "cases": cases, "rows": p.rows, "scope": "controlled battle/story 20/30 frames; remake native-theme control, no new original-executable or normal-player parity"}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
