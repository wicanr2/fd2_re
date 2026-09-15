package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

type townShopE2Fixture struct {
	name, saveEnv, runEnv, saveSHA, town, secret string
	slot, anchor, scan, saleDelta                int
}

type indexedCheckpoint struct {
	name, hash string
	pix, rgba  []byte
}

func imageRGBA(img image.Image) []byte {
	bounds := img.Bounds()
	rgba := make([]byte, 0, bounds.Dx()*bounds.Dy()*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			rgba = append(rgba, c.R, c.G, c.B, c.A)
		}
	}
	return rgba
}

func indexedRGBA(frame []byte, palette color.Palette) []byte {
	rgba := make([]byte, 0, len(frame)*4)
	for _, index := range frame {
		c := color.NRGBAModel.Convert(palette[index]).(color.NRGBA)
		rgba = append(rgba, c.R, c.G, c.B, c.A)
	}
	return rgba
}

func indexedImageHash(img image.Image) (string, bool) {
	paletted, ok := img.(*image.Paletted)
	if !ok || paletted.Rect.Dx() != 320 || paletted.Rect.Dy() != 200 ||
		paletted.Stride != 320 || len(paletted.Pix) != 320*200 {
		return "", false
	}
	return fmt.Sprintf("%x", sha256.Sum256(paletted.Pix)), true
}

func originalCheckpointFrames(t *testing.T, runDir string) []indexedCheckpoint {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(runDir, "checkpoint-*.png"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("原版 checkpoint 不存在：dir=%q err=%v", runDir, err)
	}
	frames := make([]indexedCheckpoint, 0, len(paths))
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if hash, ok := indexedImageHash(img); ok {
			paletted := img.(*image.Paletted)
			frames = append(frames, indexedCheckpoint{
				name: filepath.Base(path),
				hash: hash,
				pix:  append([]byte(nil), paletted.Pix...),
				rgba: imageRGBA(img),
			})
		}
	}
	if len(frames) == 0 {
		t.Fatalf("原版 run %q 沒有 320x200 indexed checkpoint", runDir)
	}
	return frames
}

func requireOriginalFrameMatch(
	t *testing.T, label string, originals []indexedCheckpoint, frames [][]byte, palette color.Palette,
) string {
	t.Helper()
	candidates := make([]string, 0, len(frames))
	bestDiff := 320*200 + 1
	bestCheckpoint, bestHash := "", ""
	bestMinX, bestMinY, bestMaxX, bestMaxY := 0, 0, 0, 0
	for _, frame := range frames {
		if len(frame) != 320*200 {
			continue
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(frame))
		rgba := indexedRGBA(frame, palette)
		candidates = append(candidates, hash)
		for _, original := range originals {
			if hash == original.hash {
				t.Logf("%s indexed frame match: %s %s", label, original.name, hash)
				return original.name
			}
			diff := 0
			minX, minY, maxX, maxY := 320, 200, -1, -1
			for i := 0; i < len(rgba); i += 4 {
				if rgba[i] != original.rgba[i] || rgba[i+1] != original.rgba[i+1] ||
					rgba[i+2] != original.rgba[i+2] || rgba[i+3] != original.rgba[i+3] {
					diff++
					pixel := i / 4
					x, y := pixel%320, pixel/320
					if x < minX {
						minX = x
					}
					if y < minY {
						minY = y
					}
					if x > maxX {
						maxX = x
					}
					if y > maxY {
						maxY = y
					}
				}
			}
			if diff == 0 {
				rgbaHash := fmt.Sprintf("%x", sha256.Sum256(rgba))
				t.Logf("%s RGBA frame match: %s %s", label, original.name, rgbaHash)
				return original.name
			}
			if diff < bestDiff {
				bestDiff, bestCheckpoint, bestHash = diff, original.name, original.hash
				bestMinX, bestMinY, bestMaxX, bestMaxY = minX, minY, maxX, maxY
			}
		}
	}
	t.Fatalf("%s 沒有原版逐 index／RGBA 相符幀；重製候選=%v；最近原版=%s %s，可見 RGBA 差異像素=%d/%d，範圍=(%d,%d)-(%d,%d)",
		label, candidates, bestCheckpoint, bestHash, bestDiff, 320*200,
		bestMinX, bestMinY, bestMaxX, bestMaxY)
	return ""
}

func titleLoadNativeFixture(t *testing.T, path string, slot int) *Game {
	t.Helper()
	pack, err := filepath.Abs("../../generated-assets/fd2-original-b97caf22")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(
		"../../../org_game/炎龍騎士團/FLAME2", "FDOTHER.DAT",
	))
	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	town, err := loadNativeTownUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	shop, err := loadNativeShopUIAssets(shared)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{
		camp:          campaign.NewRunner(graph),
		titlePhase:    "menu",
		titleSel:      0,
		nativeClassUI: shared,
		nativeTownUI:  town,
		nativeShopUI:  shop,
	}
	attachOfficialLocale(t, g)
	t.Setenv("FD2_NATIVE_SAVE", path)
	if !g.applyTitleMenuEvent(TitleMenuDown) ||
		!g.applyTitleMenuEvent(TitleMenuConfirm) {
		t.Fatal("標題 LOAD 未由正式 menu owner 消費")
	}
	for i := 0; i < 24; i++ {
		g.applyTitleMenuEvent(TitleMenuTick)
	}
	for i := 0; i < slot; i++ {
		if !g.applyTitleSlotEvent(TitleSlotDown) {
			t.Fatalf("LOAD slot down %d 未被正式 selector 消費", i+1)
		}
	}
	if !g.applyTitleSlotEvent(TitleSlotConfirm) {
		t.Fatalf("LOAD slot %d 確認失敗：%s", slot, g.msg)
	}
	return g
}

func TestTownShopEarlyMidLateE2Receipt(t *testing.T) {
	fixtures := []townShopE2Fixture{
		{name: "early", saveEnv: "FD2_TOWN_SHOP_E2_EARLY_SAVE", runEnv: "FD2_TOWN_SHOP_E2_EARLY_RUN", saveSHA: "57a61d958830827e9485cdbe5d72b9de1eed88b9af5bb5473d465c4c1d5d8c54", town: "town_ch06", secret: "shop_ch06_secret", slot: 0, anchor: 4, scan: 0x62, saleDelta: 37},
		{name: "mid", saveEnv: "FD2_TOWN_SHOP_E2_MID_SAVE", runEnv: "FD2_TOWN_SHOP_E2_MID_RUN", saveSHA: "84c608a3431c2e21295cc8bba631b085271da11c871edf39242fb4b3852945c7", town: "town_ch13", secret: "shop_ch13_secret", slot: 3, anchor: 1, scan: 0x69, saleDelta: 37},
		{name: "late", saveEnv: "FD2_TOWN_SHOP_E2_LATE_SAVE", runEnv: "FD2_TOWN_SHOP_E2_LATE_RUN", saveSHA: "a0e5519c49b52bbeab9c3bb1cf36957c42f02877a4c218ad54611952f74d6780", town: "town_ch27", secret: "shop_ch27_secret", slot: 1, anchor: 0, scan: 0x63, saleDelta: 1},
	}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			dataHome := t.TempDir()
			t.Setenv("XDG_DATA_HOME", dataHome)
			userDataDirCached = ""
			t.Cleanup(func() { userDataDirCached = "" })
			savePath, runDir := os.Getenv(fixture.saveEnv), os.Getenv(fixture.runEnv)
			if savePath == "" || runDir == "" {
				t.Skipf("未提供 %s／%s 正式 fixture", fixture.saveEnv, fixture.runEnv)
			}
			raw, err := os.ReadFile(savePath)
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != fixture.saveSHA {
				t.Fatalf("外部存檔 SHA-256=%s，want %s", got, fixture.saveSHA)
			}
			originals := originalCheckpointFrames(t, runDir)
			g := titleLoadNativeFixture(t, savePath, fixture.slot)
			if g.camp.NodeID() != fixture.town || g.loadErr != "" {
				t.Fatalf("正常 LOAD node=%q err=%q", g.camp.NodeID(), g.loadErr)
			}
			townFrames := make([][]byte, 0, 4)
			for pulse := 0; pulse < 4; pulse++ {
				g.nativeTownUIPulse = pulse
				frame, ok := g.composeNativeTownFrame()
				if !ok {
					t.Fatalf("town pulse %d 無法合成", pulse)
				}
				townFrames = append(townFrames, frame)
			}
			requireOriginalFrameMatch(t, fixture.name+" town", originals, townFrames, g.nativeClassUI.palette)

			for g.campSel != 1 {
				if !g.moveNativeTownSelection(1) {
					t.Fatalf("無法走到武器店：selection=%d", g.campSel)
				}
			}
			g.camp.Advance("opt1")
			g.enterNode()
			if g.nativeShopMode != "menu" || g.loadErr != "" {
				t.Fatalf("武器店 mode=%q node=%q err=%q", g.nativeShopMode, g.camp.NodeID(), g.loadErr)
			}
			initialGold := g.gold
			unitID := g.partyJoinOrder[0]
			initialCount := len(g.partyRoster[unitID].Inventory)
			g.nativeShopUIJob = nil
			if !g.setupNativeShopSellRoster() || !g.setupNativeShopSellItems() {
				t.Fatal("正式 sell roster／item owner 無法建立")
			}
			g.nativeShopMode, g.nativeShopSellConfirmSel = "sell_confirm", 0
			if !g.beginNativeShopSellSuccess() || g.nativeShopUIJob == nil {
				t.Fatal("正式 sell success owner 無法建立")
			}
			for i := 0; i < 2; i++ {
				after := g.nativeShopUIJob.after
				g.nativeShopUIJob = nil
				if after == nil {
					t.Fatalf("sell callback %d 缺失", i)
				}
				after()
			}
			if g.gold-initialGold != fixture.saleDelta ||
				len(g.partyRoster[unitID].Inventory) != initialCount-1 {
				t.Fatalf("sell delta gold=%d inventory=%d→%d", g.gold-initialGold, initialCount, len(g.partyRoster[unitID].Inventory))
			}
			g.saveGameToSlot(fixture.slot)
			if _, err := os.Stat(saveSlotPath(fixture.slot)); err != nil {
				t.Fatalf("重製 slot %d 存檔邊界失敗：%v", fixture.slot, err)
			}

			g.leaveShop()
			chapter := strings.TrimPrefix(fixture.town, "town_")
			for selection, expected := range map[int]string{
				2: "preparation_" + chapter,
				3: "shop_" + chapter + "_item",
				4: "church_" + chapter,
			} {
				for g.campSel != selection {
					if !g.moveNativeTownSelection(1) {
						t.Fatalf("無法走到城鎮選項 %d", selection)
					}
				}
				g.camp.Advance(fmt.Sprintf("opt%d", selection))
				g.enterNode()
				if g.camp.NodeID() != expected || g.loadErr != "" {
					t.Fatalf("選項 %d node=%q want=%q err=%q", selection, g.camp.NodeID(), expected, g.loadErr)
				}
				switch selection {
				case 2:
					g.camp.Advance("cancel")
					g.enterNode()
				case 3:
					g.leaveShop()
				case 4:
					g.leaveChurch()
				}
				if g.camp.NodeID() != fixture.town || g.loadErr != "" {
					t.Fatalf("選項 %d 未返回 %s：node=%q err=%q", selection, fixture.town, g.camp.NodeID(), g.loadErr)
				}
			}
			for g.campSel != fixture.anchor {
				g.moveNativeTownSelection(1)
			}
			if !g.revealNativeTownSecret(fixture.scan) ||
				!g.camp.ConfirmNativeTownSecret(g.campSel) {
				t.Fatalf("神祕商店 gate anchor=%d scan=%02X", fixture.anchor, fixture.scan)
			}
			g.enterNode()
			if g.camp.NodeID() != fixture.secret || g.nativeShopMode != "menu" {
				t.Fatalf("secret node=%q mode=%q", g.camp.NodeID(), g.nativeShopMode)
			}
			secretFrames := make([][]byte, 0, 4)
			for pulse := 0; pulse < 4; pulse++ {
				g.nativeShopUIPulse = pulse
				frame, ok := g.composeNativeShopServiceMenu()
				if !ok {
					t.Fatalf("secret pulse %d 無法合成", pulse)
				}
				secretFrames = append(secretFrames, frame)
			}
			requireOriginalFrameMatch(t, fixture.name+" secret", originals, secretFrames, g.nativeClassUI.palette)
		})
	}
}
