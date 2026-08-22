package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

type nativeLoadSlotsUIAssets struct {
	dialogueBox fdother.LMI1Entry
	strings     *fdtxt.Strings
	font        *fdtxt.Font
	palette     color.Palette
}

func parseNativeLoadSlotShotState(value string) (int, bool) {
	selection, err := strconv.Atoi(value)
	return selection, err == nil && selection >= 0 && selection < 4
}

func loadNativeLoadSlotsUIAssets() (*nativeLoadSlotsUIAssets, error) {
	fdotherPath := nativeFDOTHERPath()
	if fdotherPath == "" {
		return nil, errors.New("native load slots UI: FDOTHER.DAT unavailable")
	}
	resource13, err := fdother.ReadResource(fdotherPath, 13)
	if err != nil {
		return nil, err
	}
	entries, err := fdother.ParseLMI1(resource13)
	if err != nil || len(entries) <= 16 ||
		entries[16].Width != 310 || entries[16].Height != 86 {
		return nil, errors.New(
			"native load slots UI: FDOTHER#13 entry16 is unavailable",
		)
	}
	base := filepath.Dir(fdotherPath)
	textRaw, err := fdother.ReadResource(filepath.Join(base, "FDTXT.DAT"), 0)
	if err != nil {
		return nil, err
	}
	strings, err := fdtxt.Parse(textRaw)
	if err != nil {
		return nil, err
	}
	fontRaw, err := fdother.ReadResource(fdotherPath, 4)
	if err != nil {
		return nil, err
	}
	font, err := fdtxt.ParseFont(fontRaw)
	if err != nil {
		return nil, err
	}
	paletteRaw, err := fdother.ReadResource(fdotherPath, 0)
	if err != nil {
		return nil, err
	}
	palette, err := fdother.ParseVGAPalette(paletteRaw)
	if err != nil {
		return nil, err
	}
	palette[0] = color.NRGBA{A: 0xff}
	return &nativeLoadSlotsUIAssets{
		dialogueBox: entries[16],
		strings:     strings,
		font:        font,
		palette:     palette,
	}, nil
}

func nativeLoadSlotMetadata() ([4]campaign.NativeLoadSlot, bool) {
	if nativeSavePath := os.Getenv("FD2_NATIVE_SAVE"); nativeSavePath != "" {
		return nativeLoadSlotMetadataFromFD2SAV(nativeSavePath)
	}
	var slots [4]campaign.NativeLoadSlot
	for slot := range slots {
		raw, err := os.ReadFile(saveSlotPath(slot))
		if errors.Is(err, os.ErrNotExist) {
			slots[slot].Empty = true
			continue
		}
		if err != nil {
			return [4]campaign.NativeLoadSlot{}, false
		}
		var data saveData
		if json.Unmarshal(raw, &data) != nil ||
			data.Chapter <= 0 || data.Chapter > 0xff {
			// Remake JSON is not the native FD2.SAV ABI. Refuse values that
			// cannot map to 0x30437's raw chapter byte.
			return [4]campaign.NativeLoadSlot{}, false
		}
		slots[slot].Chapter = byte(data.Chapter)
	}
	return slots, true
}

func nativeLoadSlotMetadataFromFD2SAV(
	path string,
) ([4]campaign.NativeLoadSlot, bool) {
	var slots [4]campaign.NativeLoadSlot
	stored, err := os.ReadFile(path)
	if err != nil {
		return slots, false
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		return slots, false
	}
	for slot := range slots {
		metadata, err := fdsave.ReadVerifiedMetadata(plain, slot)
		if err != nil {
			return [4]campaign.NativeLoadSlot{}, false
		}
		if metadata.Chapter == 0xff {
			slots[slot].Empty = true
			continue
		}
		slots[slot].Chapter = metadata.Chapter
	}
	return slots, true
}

func nativeLoadSlotConfirmable(slot int) (confirmable, native bool) {
	if slot < 0 || slot >= fdsave.SlotCount {
		return false, false
	}
	slots, ok := nativeLoadSlotMetadata()
	if !ok || slots[slot].Empty {
		return false, os.Getenv("FD2_NATIVE_SAVE") != ""
	}
	return true, os.Getenv("FD2_NATIVE_SAVE") != ""
}

// confirmTitleLoadSlot 是標題四槽 selector 的正式確認擁有者。原版槽只有在
// checksum、typed restore 與戰間節點進入全部成功後才離開 loadslots；失敗會
// 保留選槽畫面，避免看似進入遊戲卻留下部分還原狀態。
func (g *Game) confirmTitleLoadSlot(selected int) bool {
	confirmable, native := nativeLoadSlotConfirmable(selected)
	if !confirmable {
		if native {
			g.msg = "無法讀取原版存檔槽"
		} else {
			g.msg = "空的存檔槽"
		}
		return false
	}
	if native {
		if err := g.loadNativeGameFromSlot(
			os.Getenv("FD2_NATIVE_SAVE"), selected,
		); err != nil {
			g.msg = err.Error()
			return false
		}
	} else {
		g.loadGameFromSlot(selected)
	}
	g.titlePhase = ""
	return true
}

// loadNativeGameFromSlot owns the original four-slot LOAD transaction. It is
// deliberately separate from title CONTINUE/current-snapshot restore.
func (g *Game) loadNativeGameFromSlot(path string, slot int) error {
	if g.camp == nil || g.camp.C == nil {
		return errors.New("原版四槽讀檔：戰役圖尚未載入")
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("原版四槽讀檔：%w", err)
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		return fmt.Errorf("原版四槽讀檔：%w", err)
	}
	snapshot, err := fdsave.InspectChapterSlot(plain, slot)
	if err != nil {
		return fmt.Errorf("原版四槽讀檔：%w", err)
	}
	catalog, err := campaign.LoadNativeCharacterCatalog(
		assetPath("assets/data/native_character_catalog.json"),
	)
	if err != nil {
		return fmt.Errorf("原版四槽讀檔：%w", err)
	}
	gates, err := campaign.LoadNativeIntermissionGateTable(
		assetPath("assets/data/native_intermission_gate.json"),
	)
	if err != nil {
		return fmt.Errorf("原版四槽讀檔：%w", err)
	}
	plan, err := campaign.BuildNativeChapterSlotRestorePlan(
		snapshot, catalog, gates, g.camp.C,
	)
	if err != nil {
		return err
	}
	if uint64(plan.Currency) > uint64(^uint(0)>>1) {
		return fmt.Errorf(
			"原版四槽讀檔：金幣 %d 超出目前平台整數範圍",
			plan.Currency,
		)
	}

	// Every fallible decode/validation step has completed. Apply the complete
	// transaction at once so malformed native data cannot leave mixed state.
	g.captureNativeMapHUDPersistence()
	runner := campaign.NewRunner(g.camp.C)
	runner.Cur = plan.EntryNode
	g.camp = runner
	g.gold = int(plan.Currency)
	g.items = nil
	g.partyMembers = plan.PartyMembers
	g.partyJoinOrder = plan.PartyJoinOrder
	g.partyRoster = plan.PartyRoster
	g.partyDeploy = map[int]bool{}
	g.handlerChapter = plan.NativeChapterIndex
	g.restoreNativeMapHUDGateA(plan.HUDGateA)
	// Native chapter slots restore persistent records and metadata, not the
	// previous battle array. Drop it before enterNode so its old gate A cannot
	// overwrite the slot byte through the cross-node capture hook.
	g.dialog, g.st, g.sel = nil, nil, nil
	g.nativeChapterRestore = &plan
	g.loadErr = ""
	g.enterNode()
	if g.loadErr != "" {
		return fmt.Errorf("原版四槽讀檔：進入 %s 失敗：%s", plan.EntryNode, g.loadErr)
	}
	g.msg = fmt.Sprintf(
		"已讀取原版槽位%d（第%d章：%s）",
		slot+1, plan.DisplayChapter, plan.EntryNode,
	)
	return nil
}

func (g *Game) drawNativeLoadSlots(screen *ebiten.Image) bool {
	assets := g.nativeLoadSlotsUI
	if assets == nil {
		return false
	}
	slots, ok := nativeLoadSlotMetadata()
	if !ok {
		return false
	}
	frame, err := campaign.ComposeNativeLoadSlotsFrame(
		make([]byte, campaign.NativeLoadSlotWidth*campaign.NativeLoadSlotHeight),
		assets.dialogueBox, assets.strings, assets.font,
		slots, g.titleSlotSel,
	)
	if err != nil {
		return false
	}
	paletted := image.NewPaletted(
		image.Rect(
			0, 0,
			campaign.NativeLoadSlotWidth, campaign.NativeLoadSlotHeight,
		),
		assets.palette,
	)
	copy(paletted.Pix, frame)
	native := ebiten.NewImageFromImage(paletted)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(native, op)
	return true
}
