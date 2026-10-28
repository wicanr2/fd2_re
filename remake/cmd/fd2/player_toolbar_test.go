package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestPlayerThemeToolbarRealInputGPU(t *testing.T) {
	if os.Getenv("FD2_TOOLBAR_INPUT_GPU") != "1" {
		t.Skip("dedicated real input process")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	g.modernStoryPortraits = nil
	// Match loadGame: UI mode is resolved once, then artwork stays independent.
	g.interfaceMode = playerInterfaceHD
	g.modernThemeCache = set
	variant := 0
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{"town": {Type: "town", NativeTownVariant: &variant}}})
	g.nativeShopMode = ""
	g.nativeShopUIJob = nil
	out, driver := os.Getenv("FD2_TEST_TOOLBAR_OUT"), os.Getenv("FD2_TEST_XTEST_INPUT")
	if out == "" || driver == "" {
		t.Fatal("explicit output and tracked input binary required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	// Each input waits for Game.Update and its final GPU capture. Batch delays
	// cannot prove that repeated F2/F4 edges and all locale frames were consumed.
	q := &floatingToolbarProbe{p: &modernHDPresentation{Game: g, toolbar: true, faithfulWidth: 1280, faithfulHeight: 848}, driver: driver, out: out, done: make(chan error, 1), steps: []floatingToolbarStep{
		{input: "Click=106,12@640x424:400", locale: "zh-Hant", hd: true, visible: true, help: false},
		{input: "Click=106,12@640x424:400", locale: "zh-Hant", hd: true, visible: true, help: false},
		{input: "F2:400", locale: "zh-Hant", hd: false, visible: true, help: false},
		{input: "Click=106,12@640x424:400", locale: "zh-Hant", hd: true, visible: true, help: false},
		{input: "Click=43,12@640x424:400", locale: "zh-Hant", hd: false, visible: true, help: false},
		{input: "Click=43,12@640x424:400", locale: "zh-Hant", hd: false, visible: true, help: false},
		{input: "F2:400", locale: "zh-Hant", hd: true, visible: true, help: false},
		{input: "F4:400", locale: "zh-Hans", hd: true, visible: true, help: false},
		{input: "F4:400", locale: "ja", hd: true, visible: true, help: false},
		{input: "F4:400", locale: "en", hd: true, visible: true, help: false},
	}}
	for i := range q.steps {
		q.steps[i].ui = playerInterfaceHD
	}
	ebiten.SetWindowSize(1920, 1272)
	ebiten.SetWindowTitle("FD2 Theme Toolbar Input")
	if err := ebiten.RunGame(q); err != nil {
		t.Fatal(err)
	}
	if q.err != nil {
		t.Fatal(q.err)
	}
	locales := map[string]bool{}
	for _, c := range q.checks {
		locales[c["locale"].(string)] = true
	}
	if !reflect.DeepEqual(q.changes, []bool{true, false, true, false, true}) || len(locales) != 4 || len(q.checks) != 10 {
		t.Fatalf("changes=%v locales=%v checks=%d", q.changes, locales, len(q.checks))
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-TOOLBAR-REAL-INPUT", "checks": q.checks, "theme_changes": q.changes, "locales": locales, "updates": q.updates, "scope": "controlled fixed toolbar town; individually acknowledged real clicks/F2/F4 and full final GPU; not full normal campaign"}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
