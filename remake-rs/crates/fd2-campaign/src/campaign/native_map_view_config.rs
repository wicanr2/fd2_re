//! Go 來源：`remake/internal/campaign/native_map_view_config.go`（50 行）。

use super::*;

impl NativeMapViewConfig {
    /// Validate 檢查節點常數的入口視圖契約。
    ///
    /// 節點常數與執行期狀態是兩種東西，契約也不同：
    ///
    ///   - 執行期的 battle.NativeMapViewState 跟著原版的寫入端走。原版只夾鏡頭與
    ///     絕對游標（那兩組的界線就寫在 0x11B48 家族的分支條件裡），可見游標
    ///     `[0x53AB9]`／`[0x53ABD]` 是自由的 dword，走行捲動會把它寫到視窗外
    ///     （收據 docs/data/ui-traces/fd2-story-pan-cursor-20260909.json 的
    ///     frames idx=75：15 格之後 visible_y = -1）。
    ///   - 節點常數描述「進場當下就要畫出來」的靜止視圖。游標框與指令環
    ///     （0x1741C 以 visible*24 定位）立刻消費可見游標，所以它必須落在 13×8
    ///     視窗內；而且章節重設 0x205DA 把六個全域一起歸零之後，鍵盤游標、走行
    ///     步進與劇情 pan 每一次寫入都同時維持 `visible = cursor - camera`，唯一
    ///     打破它的 0x149F8 是玩家在戰鬥中確認移動的執行期路徑，不會是節點的進場
    ///     常數。受版控的 13 筆節點視圖全部滿足這兩條。
    ///
    /// Go：`native_map_view_config.go:24`，`func (c NativeMapViewConfig) Validate(width, height int) error`
    pub fn validate(self, width: i64, height: i64) -> Result<()> {
        use fdother::{NATIVE_MAP_VIEWPORT_COLUMNS as COLS, NATIVE_MAP_VIEWPORT_ROWS as ROWS};
        let c = self;
        if width < COLS || height < ROWS {
            return Err(campaign_err!("campaign: 場地 {width}x{height} 小於 {COLS}x{ROWS} 視窗"));
        }
        if c.camera_x < 0 || c.camera_x > width - COLS || c.camera_y < 0 || c.camera_y > height - ROWS {
            return Err(campaign_err!(
                "campaign: 入口鏡頭 ({},{}) 不在 0..{}／0..{}",
                c.camera_x,
                c.camera_y,
                width - COLS,
                height - ROWS
            ));
        }
        if c.cursor_x < 0 || c.cursor_x >= width || c.cursor_y < 0 || c.cursor_y >= height {
            return Err(campaign_err!(
                "campaign: 入口游標 ({},{}) 不在 {width}x{height} 場地內",
                c.cursor_x,
                c.cursor_y
            ));
        }
        if c.visible_cursor_x < 0 || c.visible_cursor_x >= COLS || c.visible_cursor_y < 0 || c.visible_cursor_y >= ROWS
        {
            return Err(campaign_err!(
                "campaign: 入口可見游標 ({},{}) 不在 {COLS}x{ROWS} 視窗內",
                c.visible_cursor_x,
                c.visible_cursor_y
            ));
        }
        if c.visible_cursor_x != c.cursor_x.wrapping_sub(c.camera_x)
            || c.visible_cursor_y != c.cursor_y.wrapping_sub(c.camera_y)
        {
            return Err(campaign_err!(
                "campaign: 入口可見游標 ({},{}) 與 cursor-camera ({},{}) 不一致",
                c.visible_cursor_x,
                c.visible_cursor_y,
                c.cursor_x.wrapping_sub(c.camera_x),
                c.cursor_y.wrapping_sub(c.camera_y)
            ));
        }
        Ok(())
    }
}

// 測試：Go `remake/internal/campaign/native_map_view_config_test.go` 逐條移植。
#[cfg(test)]
mod native_map_view_config_test;
