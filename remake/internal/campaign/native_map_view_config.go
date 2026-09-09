package campaign

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// Validate 檢查節點常數的入口視圖契約。
//
// 節點常數與執行期狀態是兩種東西，契約也不同：
//
//   - 執行期的 battle.NativeMapViewState 跟著原版的寫入端走。原版只夾鏡頭與
//     絕對游標（那兩組的界線就寫在 0x11B48 家族的分支條件裡），可見游標
//     `[0x53AB9]`／`[0x53ABD]` 是自由的 dword，走行捲動會把它寫到視窗外
//     （收據 docs/data/ui-traces/fd2-story-pan-cursor-20260909.json 的
//     frames idx=75：15 格之後 visible_y = -1）。
//   - 節點常數描述「進場當下就要畫出來」的靜止視圖。游標框與指令環
//     （0x1741C 以 visible*24 定位）立刻消費可見游標，所以它必須落在 13×8
//     視窗內；而且章節重設 0x205DA 把六個全域一起歸零之後，鍵盤游標、走行
//     步進與劇情 pan 每一次寫入都同時維持 `visible = cursor - camera`，唯一
//     打破它的 0x149F8 是玩家在戰鬥中確認移動的執行期路徑，不會是節點的進場
//     常數。受版控的 13 筆節點視圖全部滿足這兩條。
func (c NativeMapViewConfig) Validate(width, height int) error {
	if width < fdother.NativeMapViewportColumns || height < fdother.NativeMapViewportRows {
		return fmt.Errorf("campaign: 場地 %dx%d 小於 %dx%d 視窗",
			width, height, fdother.NativeMapViewportColumns, fdother.NativeMapViewportRows)
	}
	if c.CameraX < 0 || c.CameraX > width-fdother.NativeMapViewportColumns ||
		c.CameraY < 0 || c.CameraY > height-fdother.NativeMapViewportRows {
		return fmt.Errorf("campaign: 入口鏡頭 (%d,%d) 不在 0..%d／0..%d",
			c.CameraX, c.CameraY,
			width-fdother.NativeMapViewportColumns, height-fdother.NativeMapViewportRows)
	}
	if c.CursorX < 0 || c.CursorX >= width || c.CursorY < 0 || c.CursorY >= height {
		return fmt.Errorf("campaign: 入口游標 (%d,%d) 不在 %dx%d 場地內",
			c.CursorX, c.CursorY, width, height)
	}
	if c.VisibleCursorX < 0 || c.VisibleCursorX >= fdother.NativeMapViewportColumns ||
		c.VisibleCursorY < 0 || c.VisibleCursorY >= fdother.NativeMapViewportRows {
		return fmt.Errorf("campaign: 入口可見游標 (%d,%d) 不在 %dx%d 視窗內",
			c.VisibleCursorX, c.VisibleCursorY,
			fdother.NativeMapViewportColumns, fdother.NativeMapViewportRows)
	}
	if c.VisibleCursorX != c.CursorX-c.CameraX || c.VisibleCursorY != c.CursorY-c.CameraY {
		return fmt.Errorf("campaign: 入口可見游標 (%d,%d) 與 cursor-camera (%d,%d) 不一致",
			c.VisibleCursorX, c.VisibleCursorY, c.CursorX-c.CameraX, c.CursorY-c.CameraY)
	}
	return nil
}
