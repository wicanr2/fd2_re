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

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestCampaignTownHotelRawRouteReturnTrace(t *testing.T) {
	c := &campaign.Campaign{
		Start: "town_ch02",
		Flags: map[string]bool{},
		Nodes: map[string]*campaign.Node{
			"town_ch02":  {Type: "town", Options: []campaign.Option{{Label: "旅館", To: "hotel_ch02"}}},
			"hotel_ch02": {Type: "hotel", Text: "旅館／整備", Next: "town_ch02"},
		},
	}
	g := &Game{camp: campaign.NewRunner(c), st: &battle.State{}}
	g.stepCampaignMenu(campaign.MenuConfirm)
	g.camp.Advance("opt0")
	g.enterNode()
	if g.camp.NodeID() != "hotel_ch02" || g.st != nil || g.hotelSel != 0 || g.hotelHasRoute {
		t.Fatalf("town→hotel boundary=%q st=%v sel=%d route=%v", g.camp.NodeID(), g.st, g.hotelSel, g.hotelHasRoute)
	}
	for selector, want := range map[byte]uint32{0: 0x2ffa5, 1: 0x30012, 2: 0x301f4, 3: 0x19953} {
		if !g.applyHotelServiceSelection(selector) || !g.hotelHasRoute || g.hotelRoute != (fdother.NativeHotelServiceRoute{Selector: selector, ResourceID: 13, Primary: want, Secondary: map[byte]uint32{3: 0x197e5}[selector]}) {
			t.Fatalf("selector %d route=%#v msg=%q", selector, g.hotelRoute, g.msg)
		}
	}
	prior := g.hotelRoute
	if g.applyHotelServiceSelection(4) || g.hotelRoute != prior {
		t.Fatalf("invalid selector unexpectedly changed route: %#v msg=%q", g.hotelRoute, g.msg)
	}
	g.leaveHotel()
	if g.camp.NodeID() != "town_ch02" {
		t.Fatalf("hotel return node=%q", g.camp.NodeID())
	}
}

// #32 使用正式酒店input與既有四槽交易；fixture只驗E1，不稱原版玩家收據。
func hotelLoadFixtureGame() *Game {
	graph := &campaign.Campaign{Start: "hotel_current", Nodes: map[string]*campaign.Node{
		"hotel_current": {Type: "hotel", Next: "town_current"}, "town_current": {Type: "town"},
		"town_ch02":  {Type: "town", Options: []campaign.Option{{To: "hotel_ch02"}}},
		"hotel_ch02": {Type: "hotel", Next: "town_ch02"}}}
	g := &Game{camp: campaign.NewRunner(graph), gold: 99, handlerChapter: 8,
		partyMembers: map[int]bool{4: true}, nativeHotelUI: &nativeHotelUIAssets{}, nativeClassUI: &nativeClassUIAssets{}}
	g.enterNode()
	g.hotelSel = nativeHotelServiceLoad
	return g
}

func TestNativeHotelLoadPublishesAtHotelThenReturnsThroughAuthoredTown(t *testing.T) {
	path := writeNativeRestoreFixture(t, 1)
	t.Setenv("FD2_NATIVE_SAVE", path)
	g := hotelLoadFixtureGame()
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	if g.nativeHotelMode != "loadslots" || g.nativeHotelSlotSel != 0 {
		t.Fatal("服務2沒有開四槽")
	}
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	if g.nativeHotelMode != "loadslots" || g.gold != 99 {
		t.Fatal("空槽發布交易")
	}
	g.handleNativeHotelInput(nativeHotelInput{down: true})
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	if g.nativeHotelMode != "loaded" || g.camp.NodeID() != "hotel_ch02" || g.hotelSel != 2 || g.gold != 789 || !g.partyMembers[9] || g.partyMembers[4] || g.handlerChapter != 1 || g.nativeChapterSlotBaseline == nil || g.nativeChapterSlotBaseline.Slot != 1 {
		t.Fatalf("loaded=%s node=%s gold=%d party=%v", g.nativeHotelMode, g.camp.NodeID(), g.gold, g.partyMembers)
	}
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	if g.nativeHotelMode != "menu" || !g.nativeHotelPromptReturn || g.hotelSel != 2 || g.camp.NodeID() != "hotel_ch02" {
		t.Fatal("確認後沒有保持服務2酒店選單")
	}
	g.handleNativeHotelInput(nativeHotelInput{esc: true})
	if g.camp.NodeID() != "town_ch02" || g.nativeHotelMode != "" || g.nativeHotelPromptReturn {
		t.Fatal("ESC沒有沿已編寫邊返回loaded town")
	}
}

func TestNativeHotelLoadRejectsEveryPreparationGateBeforeRosterPublication(t *testing.T) {
	for _, chapter := range []byte{22, 23, 24, 27, 28, 29} {
		t.Run(string(rune('A'+chapter)), func(t *testing.T) {
			t.Setenv("FD2_NATIVE_SAVE", writeNativeRestoreFixture(t, chapter))
			g := hotelLoadFixtureGame()
			g.handleNativeHotelInput(nativeHotelInput{enter: true})
			g.handleNativeHotelInput(nativeHotelInput{down: true})
			g.handleNativeHotelInput(nativeHotelInput{enter: true})
			if g.nativeHotelMode != "load_blocked" || g.gold != 99 || g.handlerChapter != 8 || !g.partyMembers[4] || g.nativeChapterSlotBaseline != nil || g.camp.NodeID() != "hotel_current" {
				t.Fatal("gate1槽沒有零交易拒收")
			}
			g.handleNativeHotelInput(nativeHotelInput{enter: true})
			if g.nativeHotelMode != "loadslots" || g.nativeHotelSlotSel != 1 {
				t.Fatal("拒絕提示沒有回原槽列表")
			}
			g.handleNativeHotelInput(nativeHotelInput{esc: true})
			if g.nativeHotelMode != "menu" || g.hotelSel != 2 || g.camp.NodeID() != "hotel_current" {
				t.Fatal("取消沒有回原酒店")
			}
		})
	}
}

func TestNativeHotelLoadRejectsTamperAndMissingAuthoredReturn(t *testing.T) {
	for _, bad := range []string{"checksum", "return-edge"} {
		t.Run(bad, func(t *testing.T) {
			path := writeNativeRestoreFixture(t, 1)
			g := hotelLoadFixtureGame()
			if bad == "checksum" {
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				raw[0x123] ^= 1
				if err := os.WriteFile(path, raw, 0644); err != nil {
					t.Fatal(err)
				}
			} else {
				g.camp.C.Nodes["town_ch02"].Options = nil
			}
			t.Setenv("FD2_NATIVE_SAVE", path)
			g.handleNativeHotelInput(nativeHotelInput{enter: true})
			g.handleNativeHotelInput(nativeHotelInput{down: true})
			g.handleNativeHotelInput(nativeHotelInput{enter: true})
			if g.nativeHotelMode != "loadslots" || g.gold != 99 || g.handlerChapter != 8 || !g.partyMembers[4] || g.nativeChapterSlotBaseline != nil || g.camp.NodeID() != "hotel_current" {
				t.Fatal("壞來源沒有原子拒收")
			}
		})
	}
}

func TestNativeHotelLoadUsesJSONBoundaryReader(t *testing.T) {
	t.Setenv("FD2_NATIVE_SAVE", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := hotelLoadFixtureGame()
	catalog, err := loadOfficialLocale("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	g.localeCatalog = catalog
	d := saveData{Node: "town_ch02", Chapter: 1, Gold: 543, Flags: map[string]bool{"keep": true}}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	p := saveSlotPath(2)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, raw, 0644); err != nil {
		t.Fatal(err)
	}
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	g.handleNativeHotelInput(nativeHotelInput{down: true})
	g.handleNativeHotelInput(nativeHotelInput{down: true})
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	if g.nativeHotelMode != "loaded" || g.camp.NodeID() != "hotel_ch02" || g.gold != 543 || !g.camp.Flags["keep"] {
		t.Fatalf("JSON邊界未回相應酒店 mode=%s node=%s gold=%d flags=%v msg=%s loadErr=%s", g.nativeHotelMode, g.camp.NodeID(), g.gold, g.camp.Flags, g.msg, g.loadErr)
	}
}

func TestAuthoredHotelRumorsReturnToSameHotelAndKeepFlags(t *testing.T) {
	graph, err := loadPlayerCampaign(canonicalCampaignReference)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for id, n := range graph.Nodes {
		if n.Type != "hotel" || n.Rumor == "" {
			continue
		}
		count++
		rumor := graph.Nodes[n.Rumor]
		if rumor == nil || rumor.Type != "story" || rumor.Next != id || len(rumor.SetFlags) == 0 {
			t.Fatalf("傳聞返回或flags錯誤 hotel=%s", id)
		}
		r := campaign.NewRunner(graph)
		r.Cur = id
		r.Advance("rumor")
		if r.NodeID() != n.Rumor {
			t.Fatal("傳聞入口錯誤")
		}
		r.Advance("")
		if r.NodeID() != id {
			t.Fatal("傳聞返回城鎮")
		}
		for flag, value := range rumor.SetFlags {
			if r.Flags[flag] != value {
				t.Fatalf("傳聞旗標未保留：%s", flag)
			}
		}
	}
	if count != 23 {
		t.Fatalf("rumor count=%d", count)
	}
}

// 同源LOAD與正式酒店input／composer探針；城鎮酒店邊由authored opt0推进，非全程GUI鍵盤E2。
func TestNativeHotelOriginalLoadSameSourceProbe(t *testing.T) {
	path := os.Getenv("FD2_HOTEL_PROBE_SAVE")
	if path == "" {
		t.Skip("未提供#32固定槽與同源原版影格")
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(stored)) != "ee9e412c0df245672e1644689f922cecb01e1b3abfd97a3d3334081921a963e5" {
		t.Fatal("#32 input hash不符")
	}
	t.Setenv("FD2_NATIVE_SAVE", path)
	t.Setenv("FD2_CAMPAIGN", canonicalCampaignReference)
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if g.camp == nil {
		t.Fatal("正式戰役未載入")
	}
	if !g.confirmTitleLoadSlot(0) || g.camp.NodeID() != "town_ch04" {
		t.Fatalf("LOAD入口=%s err=%s", g.camp.NodeID(), g.loadErr)
	}
	g.camp.Advance("opt0")
	g.enterNode()
	if g.camp.NodeID() != "hotel_ch04" || g.nativeHotelMode != "menu" {
		t.Fatal("authored酒店入口失敗")
	}
	for i := 0; i < 2; i++ {
		g.handleNativeHotelInput(nativeHotelInput{delta: 1})
	}
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	prefix := os.Getenv("FD2_HOTEL_PROBE_PREFIX")
	if prefix == "" {
		t.Fatal("缺output prefix")
	}
	dump := func(name string) {
		for portrait := range g.nativeHotelUI.portraits {
			for pulse := 0; pulse < 4; pulse++ {
				g.nativeShopUIPulse = pulse
				pixels, ok := g.composeNativeHotelFrame(portrait)
				if !ok {
					t.Fatalf("%s composer失敗", name)
				}
				pic := image.NewPaletted(image.Rect(0, 0, 320, 200), g.nativeClassUI.palette)
				copy(pic.Pix, pixels)
				file, err := os.Create(fmt.Sprintf("%s-%s-p%d-c%d.png", prefix, name, portrait, pulse))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, pic)
				closeErr := file.Close()
				if err != nil {
					t.Fatal(err)
				}
				if closeErr != nil {
					t.Fatal(closeErr)
				}
			}
		}
	}
	dump("slots")
	membersBefore := len(g.partyMembers)
	goldBefore := g.gold
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	if g.nativeHotelMode != "loaded" || g.camp.NodeID() != "hotel_ch04" || g.hotelSel != 2 || len(g.partyMembers) != membersBefore || g.gold != goldBefore {
		t.Fatal("有效槽交易或酒店停點失敗")
	}
	dump("loaded")
	g.handleNativeHotelInput(nativeHotelInput{enter: true})
	if g.nativeHotelMode != "menu" || g.hotelSel != 2 || !g.nativeHotelPromptReturn {
		t.Fatal("1DE確認後未留在服務2")
	}
	dump("menu")
	g.handleNativeHotelInput(nativeHotelInput{esc: true})
	if g.camp.NodeID() != "town_ch04" {
		t.Fatal("ESC沒有到loaded town")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	receipt := map[string]any{
		"schema_version": 1, "kind": "fd2_hotel_load_runtime_probe", "status": "passed",
		"campaign": canonicalCampaignReference, "slot_sha256": fmt.Sprintf("%x", sha256.Sum256(stored)),
		"save_unchanged":   sha256.Sum256(after) == sha256.Sum256(stored),
		"nodes":            []string{"town_ch04", "hotel_ch04", "hotel_ch04", "hotel_ch04", "town_ch04"},
		"modes":            []string{"menu", "loadslots", "loaded", "menu"},
		"selected_service": nativeHotelServiceLoad, "gold": g.gold, "party_members": len(g.partyMembers),
		"phase_method": "列舉正式頭像與游標合法相位；未同步兩側時間",
		"limit":        "城鎮酒店邊以authored opt0推進；非全程GUI鍵盤E2",
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prefix+".json", append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
