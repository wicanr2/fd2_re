package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func nativeCurrentSavePath() string {
	// Writes require an explicit writable overlay. Never infer the pristine
	// original directory from FDOTHER and overwrite research input.
	return os.Getenv("FD2_NATIVE_SAVE")
}

// buildNativeCurrentSaveStored builds the complete replacement in memory. It
// performs no filesystem mutation; the confirmation owner may therefore
// preflight all raw state and indexed assets before closing the nested menu.
func (g *Game) buildNativeCurrentSaveStored() (string, []byte, error) {
	if g == nil || g.st == nil || g.sc == nil || len(g.nativeCurrentSavePlain) != fdsave.FileSize {
		return "", nil, errors.New("原版目前戰況存檔：缺少 CONTINUE raw baseline")
	}
	path := nativeCurrentSavePath()
	if path == "" {
		return "", nil, errors.New("原版目前戰況存檔：找不到 FD2.SAV 路徑")
	}
	baseline, err := fdsave.InspectCurrentSnapshot(g.nativeCurrentSavePlain)
	if err != nil {
		return "", nil, fmt.Errorf("原版目前戰況存檔：baseline：%w", err)
	}
	if g.sc.Chapter != int(baseline.Header.Chapter)+1 ||
		!g.st.HasNativeFieldControlState || len(g.st.NativeFieldControlRaw) != fdsave.CurrentFieldControlSize ||
		!g.st.HasNativeMapViewState || g.st.NativeRoundCounter < 0 || g.st.NativeRoundCounter > 0xff ||
		g.gold < 0 || uint64(g.gold) > uint64(^uint32(0)) {
		return "", nil, errors.New("原版目前戰況存檔：章節、field、view、回合或金幣來源不一致")
	}
	records, err := campaign.BuildNativeCurrentRuntimeRecords(g.st)
	if err != nil {
		return "", nil, err
	}
	joinTable, err := campaign.LoadNativeJoinConstructorTable(
		assetPath("assets/data/native_join_constructor.json"),
	)
	if err != nil {
		return "", nil, fmt.Errorf("原版目前戰況存檔：JOIN constructor：%w", err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		return "", nil, fmt.Errorf("原版目前戰況存檔：item rows：%w", err)
	}
	persistent, persistentCount, err := campaign.BuildNativeCurrentPersistentRecords(
		g.st, baseline, joinTable, itemRows,
	)
	if err != nil {
		return "", nil, err
	}
	view := g.st.NativeMapViewState
	for _, value := range []int{
		view.CameraX, view.CameraY, view.CursorX, view.CursorY,
		view.VisibleCursorX, view.VisibleCursorY,
	} {
		if value < 0 || value > 0xff {
			return "", nil, errors.New("原版目前戰況存檔：map view 超出 byte 範圍")
		}
	}
	if view.VisibleCursorX != view.CursorX-view.CameraX ||
		view.VisibleCursorY != view.CursorY-view.CameraY ||
		g.curX != view.CursorX || g.curY != view.CursorY {
		return "", nil, errors.New("原版目前戰況存檔：游標與 native view 不一致")
	}
	options := g.currentNativeSystemOptions()
	if err := options.Validate(); err != nil {
		return "", nil, fmt.Errorf("原版目前戰況存檔：系統設定：%w", err)
	}

	snapshot := baseline
	copy(snapshot.NativeFieldControl[:], g.st.NativeFieldControlRaw)
	snapshot.NativeEventState = g.st.NativeEventState
	snapshot.RuntimeRecords = records
	snapshot.PersistentRecords = persistent
	snapshot.Header.TurnCounter = byte(g.st.NativeRoundCounter)
	snapshot.Header.RuntimeCount = byte(len(records))
	snapshot.Header.PersistentCount = persistentCount
	snapshot.Header.CameraX = byte(view.CameraX)
	snapshot.Header.CameraY = byte(view.CameraY)
	snapshot.Header.CursorX = byte(view.CursorX)
	snapshot.Header.CursorY = byte(view.CursorY)
	snapshot.Header.VisibleCursorX = byte(view.VisibleCursorX)
	snapshot.Header.VisibleCursorY = byte(view.VisibleCursorY)
	snapshot.Header.Currency = uint32(g.gold)
	snapshot.Header.Raw53AF9 = options.Raw53AF9
	snapshot.Header.HUDGateA = options.Raw51AAB
	snapshot.Header.Raw51E61 = options.Raw51E61
	snapshot.Header.Raw51E62 = options.Raw51E62
	plain, err := fdsave.WriteCurrentSnapshot(g.nativeCurrentSavePlain, snapshot)
	if err != nil {
		return "", nil, fmt.Errorf("原版目前戰況存檔：寫入 plaintext：%w", err)
	}
	stored, err := fdsave.Encode(plain)
	if err != nil {
		return "", nil, fmt.Errorf("原版目前戰況存檔：編碼：%w", err)
	}
	return path, stored, nil
}

func replaceNativeCurrentSaveAtomic(path string, stored []byte) error {
	if path == "" || len(stored) != fdsave.FileSize {
		return errors.New("原版目前戰況存檔：原子寫入輸入無效")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("原版目前戰況存檔：讀取目標屬性：%w", err)
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".FD2.SAV.tmp-*")
	if err != nil {
		return fmt.Errorf("原版目前戰況存檔：建立暫存檔：%w", err)
	}
	tmpPath := tmp.Name()
	remove := true
	defer func() {
		_ = tmp.Close()
		if remove {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		return fmt.Errorf("原版目前戰況存檔：套用權限：%w", err)
	}
	if _, err := tmp.Write(stored); err != nil {
		return fmt.Errorf("原版目前戰況存檔：寫入暫存檔：%w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("原版目前戰況存檔：同步暫存檔：%w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("原版目前戰況存檔：關閉暫存檔：%w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("原版目前戰況存檔：取代目標：%w", err)
	}
	remove = false
	if handle, err := os.Open(dir); err == nil {
		_ = handle.Sync()
		_ = handle.Close()
	}
	return nil
}
