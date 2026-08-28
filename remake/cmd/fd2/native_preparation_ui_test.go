package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestComposeNativePreparationFrameUsesRawRosterSelectors(t *testing.T) {
	base := "../../org_game/炎龍騎士團/FLAME2"
	assets, err := fdother.DecodeNativePreparationAssetsArchive(
		filepath.Join(base, "FDOTHER.DAT"),
		filepath.Join(base, "FDICON.B24"),
	)
	if err != nil {
		t.Skipf("player-provided original assets are absent: %v", err)
	}
	status, err := battle.LoadNativeItemPanelDataAssets("../../generated-assets/fd2-original-b97caf22")
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := battle.LoadScenario("../../assets/scenarios/ch01.json")
	if err != nil {
		t.Fatal(err)
	}
	units := scenario.PartyUnits(nil)
	if len(units) < 3 {
		t.Fatal("chapter one party lacks three native records")
	}
	choices, err := fdother.DecodeRawCellResource(filepath.Join(base, "FDOTHER.DAT"), 2)
	if err != nil {
		t.Fatal(err)
	}
	resource5, err := fdother.ReadResource(filepath.Join(base, "FDOTHER.DAT"), 5)
	if err != nil {
		t.Fatal(err)
	}
	dialogue := make([]fdother.RawCell, 20)
	for index := 1; index <= 19; index++ {
		dialogue[index], err = fdother.ParseLMI1RawEntry(resource5, index)
		if err != nil {
			t.Fatal(err)
		}
	}
	portraits, err := dato.DecodeResource(filepath.Join(base, "DATO.DAT"), 0x4b)
	if err != nil || len(portraits) == 0 {
		t.Fatalf("DATO#75 unavailable: frames=%d err=%v", len(portraits), err)
	}
	ids := make([]int, 3)
	roster := make(map[int]battle.Unit, len(ids))
	for i, unit := range units[:3] {
		ids[i] = unit.Fig
		roster[unit.Fig] = *unit
	}
	g := &Game{
		camp: campaign.NewRunner(&campaign.Campaign{
			Start: "prep",
			Nodes: map[string]*campaign.Node{
				"prep": {Type: "preparation", PartyLimit: 15},
			},
		}),
		nativePreparationUI: &nativePreparationUIAssets{
			roster: assets, status: status, choices: choices,
			dialogue: dialogue, portrait: portraits[0],
		},
		nativeClassUI: &nativeClassUIAssets{
			choices: choices,
			strings: status.Strings,
			font:    status.Font,
		},
		prepIDs:       ids,
		prepSel:       1,
		prepLimit:     15,
		prepSelecting: true,
		partyDeploy:   map[int]bool{ids[1]: true},
		partyRoster:   roster,
	}
	g.prepSelecting = false
	g.prepPromptSource = make([]byte, 320*200)
	prompt, ok := g.composeNativePreparationPromptFrame()
	if !ok || len(prompt) != 320*200 {
		t.Fatalf("standalone native record prompt unavailable: ok=%v length=%d", ok, len(prompt))
	}
	if !g.beginNativePreparationPromptOpening() ||
		g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 10 {
		t.Fatal("standalone native record prompt opening unavailable")
	}
	g.nativeClassUIJob = nil
	g.camp.Node().Cancel = "town"
	if prompt, ok = g.composeNativePreparationPromptFrame(); !ok || len(prompt) != 320*200 {
		t.Fatalf("town native departure prompt unavailable: ok=%v length=%d", ok, len(prompt))
	}
	g.camp.Node().Cancel = ""
	closedPrompt := false
	if !g.beginNativePreparationPromptClosing(func() { closedPrompt = true }) ||
		g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 9 {
		t.Fatal("native preparation prompt closing unavailable")
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if !closedPrompt {
		t.Fatal("prompt continuation ran before source restore")
	}
	g.prepSelecting = true
	g.prepPromptSource = nil
	frame, ok := g.composeNativePreparationFrame()
	if !ok || len(frame) != 320*200 {
		t.Fatalf("native preparation frame unavailable: ok=%v length=%d", ok, len(frame))
	}
	g.prepSelecting = false
	g.prepConfirm = true
	if !g.beginNativePreparationConfirmationOpening() ||
		g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 10 {
		t.Fatal("native preparation confirmation opening lifecycle unavailable")
	}
	g.nativeClassUIJob = nil
	confirm, ok := g.composeNativePreparationConfirmationFrame()
	if !ok || len(confirm) != 320*200 {
		t.Fatalf("native preparation confirmation unavailable: ok=%v length=%d", ok, len(confirm))
	}
	if stringWords, err := status.Strings.Words(658); err != nil || len(stringWords) != 10 {
		t.Fatalf("FDTXT index 0x292 mismatch: words=%d err=%v", len(stringWords), err)
	}
	closed := false
	if !g.beginNativePreparationConfirmationClosing(func() { closed = true }) ||
		g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 9 ||
		len(g.nativeClassUIJob.restore) != 320*200 || closed {
		t.Fatal("native preparation confirmation closing lifecycle unavailable")
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if !closed {
		t.Fatal("preparation continuation ran before the restored source was presented")
	}
	g.prepConfirm = false
	g.prepSelecting = true
	g.partyRoster[ids[1]] = battle.Unit{}
	if _, ok := g.composeNativePreparationFrame(); ok {
		t.Fatal("preparation renderer guessed a missing raw FDICON selector")
	}
}
