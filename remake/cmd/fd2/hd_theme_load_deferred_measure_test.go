package main

import (
	"os"
	"sort"
	"strconv"
	"testing"
	"time"
)

// FD2_HD_LOAD_MEASURE_WORKERS 讓量測指定主題包分組載入的載入流數（只影響量測程序）。
func init() {
	if n, err := strconv.Atoi(os.Getenv("FD2_HD_LOAD_MEASURE_WORKERS")); err == nil && n > 0 {
		modernThemeLoadWorkersOverride = n
	}
}

// TestHDThemeLoadMeasureDeferredChecks 以真實 HD 包逐筆執行延後到第一次預備的戰鬥畫格解碼與
// 透明度檢查（#317）：正式包每一筆都必須通過，也就是延後不會讓任何已登錄的戰鬥圖在遊戲中
// 退回原版。另記錄全部做完要多久（等同修正前載入時花在這裡的時間）。
func TestHDThemeLoadMeasureDeferredChecks(t *testing.T) {
	out := hdLoadMeasureSetup(t)
	set, err := loadPlayerModernTheme()
	if err != nil {
		t.Fatal(err)
	}
	type key struct {
		archive  string
		resource int
	}
	var keys []key
	frames := 0
	for resource, entry := range set.battleAnimations {
		keys = append(keys, key{"FIGANI.DAT", resource})
		frames += len(entry.paths)
	}
	for resource, entry := range set.battleEffects {
		keys = append(keys, key{"FDOTHER.DAT", resource})
		frames += len(entry.paths)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].archive != keys[j].archive {
			return keys[i].archive < keys[j].archive
		}
		return keys[i].resource < keys[j].resource
	})
	start := time.Now()
	var failures []string
	for _, k := range keys {
		if err := set.verifyBattleHDResource(k.archive, k.resource); err != nil {
			failures = append(failures, err.Error())
		}
	}
	elapsed := time.Since(start)
	hdLoadWriteReceipt(t, out, "deferred-checks", map[string]any{
		"resources":      len(keys),
		"frames":         frames,
		"verify_seconds": elapsed.Seconds(),
		"failures":       failures,
	})
	if len(failures) != 0 {
		t.Fatalf("%d registered battle resources fail the deferred checks", len(failures))
	}
}
