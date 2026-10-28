package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestPlayerPosterToolbarRealInputGPU(t *testing.T) {
	if os.Getenv("FD2_POSTER_INPUT_GPU") != "1" {
		t.Skip("dedicated poster real input process")
	}
	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = oldCache })
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// Acknowledge actual press/release edges before the next XTest event.
	t.Setenv("FD2_XTEST_HOLD_MS", "250")
	g, _ := posterTownGame(t)
	if err := g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(saveSlotPath(0))
	if err != nil {
		t.Fatal(err)
	}
	p := &modernHDPresentation{Game: g, toolbar: true, floatingToolbar: true, faithfulWidth: 1280, faithfulHeight: 800}
	p.poster.settings = defaultPlayerPosters()
	p.poster.catalogPath, p.poster.packRoot = os.Getenv("FD2_TEST_FACILITY_CATALOG"), os.Getenv("FD2_TEST_FACILITY_PACK")
	value := func(v int) *int { return &v }
	q := &floatingToolbarProbe{p: p, inputAcks: true, driver: os.Getenv("FD2_TEST_XTEST_INPUT"), out: os.Getenv("FD2_TEST_TOOLBAR_OUT"), done: make(chan error, 1), steps: []floatingToolbarStep{
		{input: "Move=320,1@640x400:200", poster: "off", posterMenu: "off"},
		{input: "Click=505,12@640x400:300", poster: "off", posterMenu: "on"},
		{input: "Click=498,58@640x400:300", poster: "hd", posterMenu: "on"},
		{input: "Click=513,94@904x400:300", poster: "hd", posterMenu: "on", posterEdge: value(0)},
		{input: "Click=598,119@904x400:300", poster: "hd", posterMenu: "on", posterFade: value(300)},
		{input: "Click=535,119@904x400:300", poster: "hd", posterMenu: "on", posterFade: value(0)},
		{input: "Click=714,58@904x400:300", poster: "classic", posterMenu: "on"},
		{input: "Click=751,94@904x400:300", poster: "classic", posterMenu: "on", posterEdge: value(96)},
		{input: "Click=725,119@904x400:300", poster: "classic", posterMenu: "on", posterFade: value(1200)},
		{input: "Click=567,144@904x400:300", poster: "classic", posterMenu: "on"},
		{input: "Click=693,144@904x400:300", poster: "classic", posterMenu: "off"},
		{input: "F10:300", poster: "off", posterMenu: "off"},
		{input: "Move=320,1@640x400:200", poster: "off", posterMenu: "off"},
		{input: "F10:300", poster: "hd", posterMenu: "off"},
		{input: "F10:300", poster: "classic", posterMenu: "off"},
		{input: "F10:300", poster: "off", posterMenu: "off"},
		{input: "F10:300", poster: "hd", posterMenu: "off"},
		{input: "Click=637,12@904x400:300", poster: "hd", posterMenu: "on"},
		{input: "F4:300", locale: "zh-Hans", poster: "hd", posterMenu: "on"},
		{input: "F4:300", locale: "ja", poster: "hd", posterMenu: "on"},
		{input: "F4:300", locale: "en", poster: "hd", posterMenu: "on"},
		{input: "Click=632,94@904x400:300", locale: "en", poster: "hd", posterMenu: "on", posterEdge: value(48)},
		{input: "Click=661,119@904x400:300", locale: "en", poster: "hd", posterMenu: "on", posterFade: value(600)},
		{input: "Escape:300", locale: "en", poster: "hd", posterMenu: "off"},
		{input: "Move=452,1@904x400:200", locale: "en", poster: "hd", posterMenu: "off"},
		{input: "F8:300", locale: "en", poster: "hd", posterMenu: "off", speed: 2},
		{input: "F8:300", locale: "en", poster: "hd", posterMenu: "off", speed: 3},
		{input: "F8:300", locale: "en", poster: "hd", posterMenu: "off", speed: 4},
		{input: "F11:300", locale: "en", ui: "original", poster: "hd", posterMenu: "off"},
		{input: "F2:300", locale: "en", ui: "original", poster: "hd", posterMenu: "off"},
		{input: "F11:300", locale: "en", ui: "hd", poster: "hd", posterMenu: "off"},
		{input: "F2:300", locale: "en", ui: "hd", poster: "hd", posterMenu: "off"},
		{input: "F10:300", locale: "en", poster: "classic", posterMenu: "off"},
		{input: "F10:300", locale: "en", poster: "off", posterMenu: "off"},
		{input: "F8:300", locale: "en", poster: "off", posterMenu: "off", speed: 1},
		{input: "F4:300", locale: "zh-Hant", poster: "off", posterMenu: "off"},
	}}
	art := true
	for i := range q.steps {
		q.steps[i].visible = true
		if q.steps[i].locale == "" {
			q.steps[i].locale = "zh-Hant"
		}
		if q.steps[i].ui == "" {
			q.steps[i].ui = "hd"
		}
		if strings.HasPrefix(q.steps[i].input, "F2:") {
			art = !art
		}
		q.steps[i].hd = art
	}
	if q.driver == "" || q.out == "" || os.MkdirAll(q.out, 0755) != nil {
		t.Fatal("explicit tracked driver and output required")
	}
	ebiten.SetWindowSize(1920, 1200)
	ebiten.SetWindowTitle("FD2 Poster Toolbar Input")
	if err := ebiten.RunGame(q); err != nil {
		t.Fatal(err)
	}
	if q.err != nil || len(q.checks) != len(q.steps) {
		t.Fatal("incomplete poster real input", len(q.checks), q.err)
	}
	if err := g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(saveSlotPath(0))
	if err != nil || !bytes.Equal(saved, after) {
		t.Fatal("poster controls changed campaign save", err)
	}
	s := loadSettings()
	if s.Poster == nil || *s.Poster != defaultPlayerPosters() {
		t.Fatal("poster choices did not survive settings cold read", s.Poster)
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-POSTER-REAL-INPUT", "checks": q.checks, "physical_input": q.inputTrace, "save_bytes_unchanged": len(saved), "xtest_hold_ms": 250, "scope": "controlled town; real pointer, slider, F10/F2/F11/F4/Escape; not normal campaign or platform evidence"}, "", "  ")
	if err := os.WriteFile(filepath.Join(q.out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
