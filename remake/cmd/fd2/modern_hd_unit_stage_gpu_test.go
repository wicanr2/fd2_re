package main

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestModernHDUnitStageCompleteGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_UNIT_STAGE_GPU") != "1" {
		t.Skip("dedicated map23 unit stage Game.Draw")
	}
	id, err := strconv.Atoi(os.Getenv("FD2_HD_UNIT_STAGE_CASE"))
	if err != nil || id < 0 || id > 1 {
		t.Fatal("explicit stage case0/1 required")
	}
	out := os.Getenv("FD2_TEST_UNIT_PRESENT_OUT")
	if out == "" {
		t.Fatal("fresh capture path required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	set := itemHDSet(t)
	g, a, b := unitStageRelocationGame(t, id == 0, set)
	control, _, _ := unitStageRelocationGame(t, id == 0, nil)
	done, cdone := new(int), new(int)
	for i, game := range []*Game{g, control} {
		count := done
		if i == 1 {
			count = cdone
		}
		j := game.nativeModifierPresentation.unitPresent.hdNext
		next := j.then
		j.then = func() { *count++; next() }
	}
	dest := image.Pt(27, 14)
	if id == 0 {
		dest.Y = 19
	}
	c := unitPresentGPUCase{name: "command23", g: g, control: control, actor: a, target: b, continued: done, controlContinued: cdone,
		legs: map[*nativeUnitPresentJob]int{}, after: map[*nativeUnitPresentJob][]byte{}, f2: map[string]bool{}, shots: map[string]bool{}, finalDestination: &dest}
	p := &unitPresentGPUProbe{set: set, out: out, cases: []unitPresentGPUCase{c}}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 1 || p.source != 22 || *done != 1 || *cdone != 1 {
		t.Fatal("incomplete stage relocation", p.err, p.index, p.source, *done, *cdone)
	}
	c = p.cases[0]
	rows := make([]int, 2)
	for j, leg := range c.legs {
		rows[leg] = len(j.vgaFrames) - 27
	}
	want := []int{18, 24}
	if id == 1 {
		want = []int{24, 18}
	}
	if rows[0] != want[0] || rows[1] != want[1] || len(c.f2) != 20 || len(c.shots) != 11 || c.returnSHA == "" || g.nativeRNGState != 7 || a.InventorySlots[0] != 101 {
		t.Fatal("stage branch/return/transaction differs", rows, len(c.f2), len(c.shots), a.InventorySlots)
	}
	d := map[string]any{"status": "PASS-UNIT-STAGE-HD-GAME", "map_index": 23, "case_id": id, "bridge_rows": rows, "draws": c.draws,
		"f2_round_trips": len(c.f2), "continuations": *done, "native_control_unchanged": true, "shots": len(c.shots), "return_final_sha256": c.returnSHA,
		"sources": p.sources, "rows": p.rows, "fixed_clock_unix": 1000, "native_seed": 7, "rgba_tolerance": 2,
		"scope": "actual map23 target/destination controller with controlled item/records/placement; RUNTIME-E1, not normal-player or original oracle proof"}
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
