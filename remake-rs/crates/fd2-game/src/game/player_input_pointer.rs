//! Go 來源：`remake/cmd/fd2/player_input_pointer.go`（258 行，Go v.1.1.15 `7c710b34`）。R5 階段 4（r5i）移植。
//!
//! 滑鼠指向換算成游標格或選單列。只在 `pointer.active`（滑鼠這一刻移動或按下左鍵）時作用；換算後走的是方向鍵同一條
//! 路（`move_map_cursor`、選單的上下事件、指令環的方向閘門），所以原版的游標規則、鏡頭安全帶與不可用方向的閘門都照舊。
//! 左鍵另外由 [`PlayerInput`] 轉成 Enter：同一刻先移到指向的列，再確認。點到不能選的指令環方向時作廢這一次 Enter，
//! 避免確認到別的指令。

use super::*;
use crate::go_ebiten_keys as ek;

/// 原生地圖窗格：0x11EB0 把 312×192 搬到 VGA (4,4)，每格 24 原生像素；畫布是兩倍。
///
/// Go：`player_input_pointer.go:20`..`24`
pub(crate) const NATIVE_VIEWPORT_CANVAS_X: i64 = indexedmap::NATIVE_MAP_VIEWPORT_X0 * 2;
pub(crate) const NATIVE_VIEWPORT_CANVAS_Y: i64 = indexedmap::NATIVE_MAP_VIEWPORT_Y0 * 2;
pub(crate) const NATIVE_CELL_CANVAS: i64 = 24 * 2;

/// Go：`player_input_pointer.go:26`，`func floorDiv(a, b int) int`
pub(crate) fn floor_div(a: i64, b: i64) -> i64 {
    let mut q = a / b;
    if a % b != 0 && (a < 0) != (b < 0) {
        q -= 1;
    }
    q
}

impl Game {
    /// 指向的地圖格。原生地圖依窗格幾何；舊版地圖沿觸控點格的換算。
    ///
    /// Go：`player_input_pointer.go:35`，`func (g *Game) pointerMapCell() (int, int, bool)`
    pub(crate) fn pointer_map_cell(&self) -> (i64, i64, bool) {
        let Some(m) = self.m.as_ref().filter(|m| self.pointer.active && m.w > 0 && m.h > 0) else {
            return (0, 0, false);
        };
        let (x, y);
        if let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_view_state) {
            let col = floor_div(self.pointer.x - NATIVE_VIEWPORT_CANVAS_X, NATIVE_CELL_CANVAS);
            let row = floor_div(self.pointer.y - NATIVE_VIEWPORT_CANVAS_Y, NATIVE_CELL_CANVAS);
            if col < 0
                || row < 0
                || col >= fdother::NATIVE_MAP_VIEWPORT_COLUMNS
                || row >= fdother::NATIVE_MAP_VIEWPORT_ROWS
            {
                return (0, 0, false);
            }
            let view = &st.native_map_view_state;
            (x, y) = (view.camera_x + col, view.camera_y + row);
        } else {
            if m.tile_w <= 0 || m.tile_h <= 0 {
                return (0, 0, false);
            }
            // Go `int(g.camX)`：float64 轉 int 向零截斷。
            (x, y) = ((self.cam_x as i64 + self.pointer.x) / m.tile_w, (self.cam_y as i64 + self.pointer.y) / m.tile_h);
        }
        if x < 0 || y < 0 || x >= m.w || y >= m.h {
            return (0, 0, false);
        }
        (x, y, true)
    }

    /// 把地圖游標逐格移到指向的格子（先 X 後 Y，同觸控點格），每一步都走鍵盤游標處理器。某一步被拒絕就停下，
    /// 不會卡在迴圈裡。
    ///
    /// Go：`player_input_pointer.go:62`，`func (g *Game) followPointerOnMap()`
    pub(crate) fn follow_pointer_on_map(&mut self) {
        let (target_x, target_y, ok) = self.pointer_map_cell();
        if !ok {
            return;
        }
        let (map_w, map_h) = self.m.as_ref().map_or((0, 0), |m| (m.w, m.h));
        let mut steps = 0;
        while steps < map_w + map_h + 4 && (self.cur_x != target_x || self.cur_y != target_y) {
            let (mut dx, mut dy) = (0, 0);
            if self.cur_x < target_x {
                dx = 1;
            } else if self.cur_x > target_x {
                dx = -1;
            } else if self.cur_y < target_y {
                dy = 1;
            } else {
                dy = -1;
            }
            let (before_x, before_y) = (self.cur_x, self.cur_y);
            self.move_map_cursor(dx, dy);
            if self.cur_x == before_x && self.cur_y == before_y {
                return;
            }
            steps += 1;
        }
    }

    /// 依指向點相對可見游標格中心的主軸，回傳指令環的方向（0 上、1 左、2 右、3 下，同
    /// `apply_native_overlay_direction_input`）。四個圖格開在游標格的上下左右一格（0x1741c 的末幀位移），所以指在游標
    /// 格本身或兩格以外都不算。
    ///
    /// Go：`player_input_pointer.go:90`，`func (g *Game) pointerOverlayDirection() (int, bool)`
    pub(crate) fn pointer_overlay_direction(&self) -> (i64, bool) {
        let Some(st) = self.st.as_ref().filter(|st| self.pointer.active && st.has_native_map_view_state) else {
            return (0, false);
        };
        let view = &st.native_map_view_state;
        let center_x = NATIVE_VIEWPORT_CANVAS_X + view.visible_cursor_x * NATIVE_CELL_CANVAS + NATIVE_CELL_CANVAS / 2;
        let center_y = NATIVE_VIEWPORT_CANVAS_Y + view.visible_cursor_y * NATIVE_CELL_CANVAS + NATIVE_CELL_CANVAS / 2;
        let (dx, dy) = (self.pointer.x - center_x, self.pointer.y - center_y);
        let (ax, ay) = (dx.abs(), dy.abs());
        if ax.max(ay) < NATIVE_CELL_CANVAS / 2 || ax.max(ay) > NATIVE_CELL_CANVAS * 2 {
            return (0, false);
        }
        if ay >= ax {
            if dy < 0 {
                return (0, true);
            }
            return (3, true);
        }
        if dx < 0 {
            return (1, true);
        }
        (2, true)
    }

    /// 在方向鍵之後套用指向的方向；不可用的方向照原版閘門不改選擇。
    ///
    /// Go：`player_input_pointer.go:121`，`func (g *Game) applyPointerToOverlay(availability [4]int)`
    pub(crate) fn apply_pointer_to_overlay(&mut self, availability: [i64; 4]) {
        let (direction, ok) = self.pointer_overlay_direction();
        if ok && fdother::action_overlay_accepts_direction(availability, direction) {
            self.ring_sel = direction;
        }
    }

    /// 在指令環讀 Enter 之前判斷：點在不可用的方向上，就作廢這一下。
    ///
    /// Go：`player_input_pointer.go:128`，`func (g *Game) pointerOverlayRejectsClick()`
    pub(crate) fn pointer_overlay_rejects_click(&mut self) {
        // 只在指令環本身等方向輸入時判斷；物品欄、指令格、法術表與 END 確認各自畫在別處。
        if !self.pointer.clicked
            || !self.ring
            || self.item_open
            || self.native_command_open
            || self.spell_open
            || self.native_system_end_turn_confirm
            || self.native_system_info_ui.is_some()
        {
            return;
        }
        let (direction, ok) = self.pointer_overlay_direction();
        if !ok {
            return;
        }
        let availability = self.native_action_overlay_state().availability;
        if !fdother::action_overlay_accepts_direction(availability, direction) && self.player_input.is_some() {
            self.suppress_virtual_key(ek::KEY_ENTER);
        }
    }

    /// 標題選單三項的畫布矩形（drawTitle 的 menu 分支同一組座標）。
    ///
    /// Go：`player_input_pointer.go:144`，`func (g *Game) titleMenuRects() []image.Rectangle`
    pub(crate) fn title_menu_rects(&self) -> Vec<Rectangle> {
        let Some(ta) = self.title_assets.as_ref() else {
            return Vec::new();
        };
        let mut rects = Vec::with_capacity(3);
        for i in 0..3 {
            let Some(it) = ta.items[i][0].as_ref() else {
                return Vec::new();
            };
            let (w, h) = (it.width, it.height);
            let _ = h;
            let (x, y) = ((320 - w) / 2 * 2, title_menu_y(i as i64) * 2);
            rects.push(rect(x, y, x + w * 2, y + TITLE_MENU_GAP_Y * 2));
        }
        rects
    }

    /// 走 TitleMenuUp／Down 到指向的項目；確認閃爍中不動。
    ///
    /// Go：`player_input_pointer.go:163`，`func (g *Game) applyPointerToTitleMenu()`
    pub(crate) fn apply_pointer_to_title_menu(&mut self) {
        if !self.pointer.active || self.title_flash != 0 {
            return;
        }
        let target = pointer_in(&self.title_menu_rects(), self.pointer.x, self.pointer.y);
        let mut steps = 0;
        while target >= 0 && self.title_sel != target && steps < 3 {
            let mut event = TITLE_MENU_DOWN;
            if (self.title_sel + 2) % 3 == target {
                event = TITLE_MENU_UP;
            }
            self.apply_title_menu_event(event);
            steps += 1;
        }
    }

    /// Go：`player_input_pointer.go:187`，`func (g *Game) applyPointerToTitleSlots()`
    ///
    /// Go 走 `applyTitleSlotEvent`（`title.go:86`，`r8`：標題 LOAD 讀檔槽在 R8）；Rust 與鍵盤上下鍵相同，要換列時停在
    /// `todo!`。指向的列由 [`title_slot_rects`] 決定。
    pub(crate) fn apply_pointer_to_title_slots(&mut self) {
        if !self.pointer.active {
            return;
        }
        let target = pointer_in(&title_slot_rects(), self.pointer.x, self.pointer.y);
        if target >= 0 && self.title_slot_sel != target {
            todo!("R8: title.go:applyTitleSlotEvent（標題 LOAD 讀檔槽，滑鼠指向換列）")
        }
    }

    /// 沿 0x2cf01..0x2cf65 的左右循環走到指向的設施，取步數少的方向。
    ///
    /// Go：`player_input_pointer.go:216`，`func (g *Game) applyPointerToNativeTown(variant int)`
    pub(crate) fn apply_pointer_to_native_town(&mut self, variant: i64) {
        if !self.pointer.active || self.camp_sel < 0 || self.camp_sel > 4 {
            return;
        }
        let target = pointer_in(&native_town_selection_rects(variant), self.pointer.x, self.pointer.y);
        if target < 0 || target == self.camp_sel {
            return;
        }
        let left = (target - self.camp_sel + 5) % 5; // moveNativeTownSelection(1) 的步數
        let mut delta = 1;
        if 5 - left < left {
            delta = -1;
        }
        let mut steps = 0;
        while self.camp_sel != target && steps < 5 {
            if !self.move_native_town_selection(delta) {
                return;
            }
            steps += 1;
        }
    }

    /// Go：`player_input_pointer.go:245`，`func (g *Game) applyPointerToChoiceMenu()`
    pub(crate) fn apply_pointer_to_choice_menu(&mut self) {
        let Some(camp) = self.camp.as_ref().filter(|_| self.pointer.active) else {
            return;
        };
        let count = camp.visible().len() as i64;
        let target = pointer_in(&choice_menu_rects(count), self.pointer.x, self.pointer.y);
        let mut steps = 0;
        while target >= 0 && self.camp_sel != target && steps < count {
            let mut event = campaign::MENU_DOWN;
            if target < self.camp_sel {
                event = campaign::MENU_UP;
            }
            self.step_campaign_menu(event);
            steps += 1;
        }
    }
}

/// 讀檔四格的列位置沿 `campaign.ComposeNativeLoadSlotsFrame`：文字列 y=119+19×列、框寬 5..315。
///
/// Go：`player_input_pointer.go:178`，`func titleSlotRects() []image.Rectangle`
pub(crate) fn title_slot_rects() -> Vec<Rectangle> {
    (0..4)
        .map(|row| {
            let y = 119 + row * 19;
            rect(5 * 2, (y - 2) * 2, 315 * 2, (y + 17) * 2)
        })
        .collect()
}

/// 五個設施游標人物的位置（每格 24 原生像素，四周各放寬 4）。
///
/// Go：`player_input_pointer.go:202`，`func nativeTownSelectionRects(variant int) []image.Rectangle`
pub(crate) fn native_town_selection_rects(variant: i64) -> Vec<Rectangle> {
    let mut rects = Vec::with_capacity(5);
    for selection in 0..5 {
        let Ok(geometry) = campaign::native_town_cursor_geometry_for(variant, selection, 0) else {
            return Vec::new();
        };
        let p = geometry.position;
        rects.push(rect((p.x - 4) * 2, (p.y - 4) * 2, (p.x + 28) * 2, (p.y + 28) * 2));
    }
    rects
}

/// 重製端選項框的列（drawCampaign 的 choice 分支：x 160..480、第 i 列 y=162+28i）。
///
/// Go：`player_input_pointer.go:237`，`func choiceMenuRects(count int) []image.Rectangle`
pub(crate) fn choice_menu_rects(count: i64) -> Vec<Rectangle> {
    (0..count.max(0)).map(|i| rect(160, 158 + i * 28, 480, 186 + i * 28)).collect()
}
