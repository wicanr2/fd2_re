//! Go 來源：`remake/cmd/fd2/native_staging_present.go`（178 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// nativeFocusEndpoint predicts the exact 0x12CEA safe-band endpoint without
    /// publishing a focus step.  sub_33F78 needs this private endpoint to preflight
    /// the following 0x22253 transaction before the first visible mutation.
    ///
    /// Go：`native_staging_present.go:15`，`func (g *Game) nativeFocusEndpoint(targetX, targetY int) (int, int, battle.NativeMapViewState, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_focus_endpoint(
        &self,
        target_x: i64,
        target_y: i64,
    ) -> Result<(i64, i64, battle::NativeMapViewState)> {
        let unavailable = || GameError::new("native 0x33F78 focus endpoint is unavailable");
        let Some(m) = self.m.as_ref() else {
            return Err(unavailable());
        };
        if m.tile_w <= 0
            || m.tile_h <= 0
            || target_x < 0
            || target_y < 0
            || target_x >= m.w
            || target_y >= m.h
            || self.cam_x as i64 % m.tile_w != 0
            || self.cam_y as i64 % m.tile_h != 0
        {
            return Err(unavailable());
        }
        let (mut origin_x, mut origin_y) = (self.cam_x as i64 / m.tile_w, self.cam_y as i64 / m.tile_h);
        let (mut cursor_x, mut cursor_y) = (self.cur_x, self.cur_y);
        // 可見游標一律沿用已追蹤的值；原版沒有任何一處由 cursor - camera 重算它
        // （寫入端見 docs/data/ida/fd2_visible_cursor_writers_ida.txt）。鏡頭捲過
        // 游標之後那個減法會給出負值，也就是視窗外的無效格。
        let (mut screen_x, mut screen_y) =
            native_map_focus_visible_seed(cursor_x.wrapping_sub(origin_x), cursor_y.wrapping_sub(origin_y));
        if self.has_story_native_map_view {
            // The six globals are one typed state. Mixing the story-visible cursor
            // with g.curX/g.curY from a preceding battle creates an impossible
            // cursor-camera identity after LOADCH and PAN.
            let v = self.story_native_map_view;
            (cursor_x, cursor_y, screen_x, screen_y) = (v.cursor_x, v.cursor_y, v.visible_cursor_x, v.visible_cursor_y);
        } else if let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_view_state) {
            let v = st.native_map_view_state;
            (cursor_x, cursor_y, screen_x, screen_y) = (v.cursor_x, v.cursor_y, v.visible_cursor_x, v.visible_cursor_y);
        }
        let (mut max_origin_x, mut max_origin_y) = (m.w - 13, m.h - 8);
        if max_origin_x < 0 {
            max_origin_x = 0;
        }
        if max_origin_y < 0 {
            max_origin_y = 0;
        }
        let mut steps = 0;
        while cursor_x != target_x || cursor_y != target_y {
            if steps > m.w + m.h + 32 {
                return Err(GameError::new("native 0x33F78 focus endpoint exceeded map bound"));
            }
            if cursor_x > target_x {
                cursor_x -= 1;
                if screen_x < 2 && origin_x > 0 {
                    origin_x -= 1;
                } else {
                    screen_x -= 1;
                }
            } else if cursor_x < target_x {
                cursor_x += 1;
                if screen_x > 10 && origin_x < max_origin_x {
                    origin_x += 1;
                } else {
                    screen_x += 1;
                }
            } else if cursor_y > target_y {
                cursor_y -= 1;
                if screen_y < 2 && origin_y > 0 {
                    origin_y -= 1;
                } else {
                    screen_y -= 1;
                }
            } else if cursor_y < target_y {
                cursor_y += 1;
                if screen_y > 5 && origin_y < max_origin_y {
                    origin_y += 1;
                } else {
                    screen_y += 1;
                }
            }
            steps += 1;
        }
        let view = battle::NativeMapViewState {
            camera_x: origin_x,
            camera_y: origin_y,
            cursor_x: target_x,
            cursor_y: target_y,
            visible_cursor_x: screen_x,
            visible_cursor_y: screen_y,
        };
        let mut carrier = battle::State::default();
        (carrier.w, carrier.h) = (m.w, m.h);
        if let Err(err) = carrier.materialize_native_map_view_state(view) {
            return Err(game_err!(
                "{err}: view={} map={}x{} start_cursor=({},{}) start_origin=({},{})",
                go_native_map_view_plus_v(view),
                m.w,
                m.h,
                self.cur_x,
                self.cur_y,
                self.cam_x as i64 / m.tile_w,
                self.cam_y as i64 / m.tile_h
            ));
        }
        Ok((origin_x * m.tile_w, origin_y * m.tile_h, carrier.native_map_view_state))
    }
}
