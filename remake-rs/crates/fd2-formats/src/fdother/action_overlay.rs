//! 移植自 Go `action_overlay.go`：FD2.EXE `0x1741c` 系列的四向動作覆蓋層。

use super::{FdotherError, RawCell, Result};

/// `0x1741c` 消費的兩組 raw 四字輸入，原生順序是上、左、右、下。`availability` 是呼叫端的第二個
/// 參數，乘以二；只有在戰鬥包裝裡才代表動作可用性。`direction_state` 是呼叫端的第一個參數，
/// 乘以三。其他呼叫端可能把任一表當成無關的選單狀態，所以兩個欄位本身都不指定圖示或玩法語意。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ActionOverlayState {
    pub availability: [i64; 4],
    pub direction_state: [i64; 4],
}

/// 建出原生戰鬥動作包裝 `0x18d8c` 傳入的狀態。`direction_state` 固定是 [0,1,2,3]；可用（零）的字
/// 選到格 [0,3,6,9]，一的字選到 [2,5,8,11]。其他 `0x1741c` 呼叫端用各自的表，必須明確建構
/// [`ActionOverlayState`]。
pub fn battle_action_overlay_state(availability: [i64; 4]) -> ActionOverlayState {
    ActionOverlayState { availability, direction_state: [0, 1, 2, 3] }
}

/// 固定參考 FD2.SAV 目前執行期 CONTINUE 錨點所用的 raw 初始 `0x16f55` 呼叫端狀態。把那張一般玩家
/// 截圖對到 `0x16f55` 是強推論；兩張表本身是直接指令證據。第一個參數表在 `0x51e9f` 是 [7,5,6,4]；
/// 第二個參數表在 `0x53ef2` 全為零，得到格 [21,15,18,12]。這裡只提供可見的覆蓋層，不替四個動作
/// 命名也不啟用它們；它們的擁有者在重製端仍是失敗即關閉。
pub fn native_continue_action_overlay_state() -> ActionOverlayState {
    ActionOverlayState { availability: [0; 4], direction_state: [7, 5, 6, 4] }
}

/// 重現 `sub_19DF7` 的巢狀戰場選單。第一張表是 [12,13,14,15]。這裡只呈現已直接證實的表寫入端：
/// raw 執行期記錄條件成立時索引 1 變成一，FD2.SAV 不存在時索引 2 變成一。格的變化本身不證明
/// 一般的啟用／停用語意；動作交易仍歸呼叫端。
pub fn native_nested_system_action_overlay_state(save_gate_set: bool, save_file_absent: bool) -> ActionOverlayState {
    let mut state = ActionOverlayState { direction_state: [12, 13, 14, 15], ..Default::default() };
    if save_gate_set {
        state.availability[1] = 1;
    }
    if save_file_absent {
        state.availability[2] = 1;
    }
    state
}

/// 保存 FD2.EXE `0x1728c` 讀寫的四個 raw 位元組。以位址命名讓這個具型別轉接層保持誠實：較高層的
/// 語意來自各自獨立證實的消費端。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeSystemOptions {
    pub raw_53af9: u8,
    pub raw_51aab: u8,
    pub raw_51e61: u8,
    pub raw_51e62: u8,
}

pub fn default_native_system_options() -> NativeSystemOptions {
    NativeSystemOptions { raw_53af9: 0, raw_51aab: 1, raw_51e61: 1, raw_51e62: 1 }
}

impl NativeSystemOptions {
    pub fn validate(&self) -> Result<()> {
        let values = [self.raw_53af9, self.raw_51aab, self.raw_51e61, self.raw_51e62];
        for (i, value) in values.into_iter().enumerate() {
            if value > 1 {
                return Err(FdotherError::message(format!("fdother: native system option {i} has raw value {value}")));
            }
        }
        Ok(())
    }

    pub fn music_enabled(&self) -> bool {
        self.raw_51e61 == 1
    }

    pub fn sfx_enabled(&self) -> bool {
        self.raw_51e62 == 1
    }

    pub fn full_presentation_enabled(&self) -> bool {
        self.raw_53af9 == 0
    }

    pub fn hud_enabled(&self) -> bool {
        self.raw_51aab == 1
    }

    /// 重現 `0x172c4..0x17318`。第二個參數表全為零，所以每個開／關格直接由第一張表表示。
    ///
    /// Go 以 `int(1-s.Raw51E61)` 這種 byte 減法再轉 int；驗證過的值只有 0／1，結果相同。
    pub fn action_overlay_state(&self) -> Result<ActionOverlayState> {
        self.validate()?;
        Ok(ActionOverlayState {
            availability: [0; 4],
            direction_state: [
                18 + (1 - self.raw_51e61) as i64,
                20 + (1 - self.raw_51e62) as i64,
                22 + self.raw_53af9 as i64,
                24 + (1 - self.raw_51aab) as i64,
            ],
        })
    }

    /// Go 版以值接收者回傳 `(s, err)`；出錯時回傳的 `s` 永遠是未改動的輸入，所以 Rust 只回傳錯誤。
    pub fn toggle(&self, selector: i64) -> Result<NativeSystemOptions> {
        self.validate()?;
        let mut s = *self;
        match selector {
            0 => s.raw_51e61 ^= 1,
            1 => s.raw_51e62 ^= 1,
            2 => s.raw_53af9 ^= 1,
            3 => s.raw_51aab ^= 1,
            _ => {
                return Err(FdotherError::message(format!(
                    "fdother: native system option selector {selector} is invalid"
                )));
            }
        }
        Ok(s)
    }
}

const NATIVE_FRAMEBUFFER_STRIDE: i64 = 0x1c8;
const NATIVE_ACTION_OVERLAY_BASE: i64 = 0x8088;
const NATIVE_ACTION_OVERLAY_STEP: i64 = 0x18;

/// `0x1741C` 把可見游標當格座標時的視窗大小。同一組常數也出現在 `0x174AE`／`0x174B0` 的
/// push 0xD／push 8。
pub const NATIVE_MAP_VIEWPORT_COLUMNS: i64 = 13;
pub const NATIVE_MAP_VIEWPORT_ROWS: i64 = 8;

/// `0x175a9`／`0x17643` 使用的私有索引備份大小。呼叫端必須明確提供左上角矩形；本套件不從游標
/// 或相對 blit 偏移推論畫面錨點。
pub const NATIVE_ACTION_OVERLAY_SNAPSHOT_WIDTH: i64 = 72;
pub const NATIVE_ACTION_OVERLAY_SNAPSHOT_HEIGHT: i64 = 72;
pub const NATIVE_ACTION_OVERLAY_SNAPSHOT_BYTES: i64 =
    NATIVE_ACTION_OVERLAY_SNAPSHOT_WIDTH * NATIVE_ACTION_OVERLAY_SNAPSHOT_HEIGHT;

/// 檢查呼叫端擁有的索引幀緩衝矩形，不替它指定任何未證實的座標語意。
///
/// `x+72`、`y+72` 與 Go 一樣以 64 位元繞回；x 或 y 接近 int 上限時繞成負值而通過檢查，之後的
/// 切片運算 panic（Go 同樣 panic）。
fn validate_action_overlay_snapshot_rect(buf: &[u8], stride: i64, x: i64, y: i64) -> Result<()> {
    if stride < NATIVE_ACTION_OVERLAY_SNAPSHOT_WIDTH
        || x < 0
        || y < 0
        || x.wrapping_add(NATIVE_ACTION_OVERLAY_SNAPSHOT_WIDTH) > stride
        || y.wrapping_add(NATIVE_ACTION_OVERLAY_SNAPSHOT_HEIGHT) > buf.len() as i64 / stride
    {
        return Err(FdotherError::fixed("fdother: action overlay snapshot rectangle is invalid"));
    }
    Ok(())
}

/// Go 切片 `buf[start:start+72]` 的列起點；乘加與 Go 一樣繞回，負值轉成極大的 `usize` 讓切片 panic。
fn snapshot_row(stride: i64, x: i64, y: i64, row: i64) -> std::ops::Range<usize> {
    let start = y.wrapping_add(row).wrapping_mul(stride).wrapping_add(x);
    let end = start.wrapping_add(NATIVE_ACTION_OVERLAY_SNAPSHOT_WIDTH);
    (start as usize)..(end as usize)
}

/// 複製原生 72×72 索引備份區。x、y 是呼叫端明確提供的左上角矩形；這裡不推論游標、鏡頭、
/// 調色盤或畫面座標語意。
pub fn capture_action_overlay_snapshot(src: &[u8], stride: i64, x: i64, y: i64) -> Result<Vec<u8>> {
    if stride <= 0 || src.is_empty() {
        return Err(FdotherError::fixed("fdother: action overlay snapshot source is invalid"));
    }
    validate_action_overlay_snapshot_rect(src, stride, x, y)?;
    let mut snapshot = vec![0u8; NATIVE_ACTION_OVERLAY_SNAPSHOT_BYTES as usize];
    let w = NATIVE_ACTION_OVERLAY_SNAPSHOT_WIDTH as usize;
    for row in 0..NATIVE_ACTION_OVERLAY_SNAPSHOT_HEIGHT {
        let at = row as usize * w;
        snapshot[at..at + w].copy_from_slice(&src[snapshot_row(stride, x, y, row)]);
    }
    Ok(snapshot)
}

/// 實作 `0x175a9`／`0x17643` 的備份位址。和 [`action_overlay_origin`] 不同，原生備份矩形從可見
/// 游標原點往左上各退一個 24 像素格。呼叫端仍擁有幀緩衝基底，不可把這個位元組位址重新解讀成
/// 畫面像素座標。
pub fn action_overlay_snapshot_origin(cursor_column: i64, cursor_row: i64) -> Result<i64> {
    if cursor_column <= 0 || cursor_row <= 0 {
        return Err(FdotherError::fixed("fdother: action overlay snapshot cursor is invalid"));
    }
    if cursor_column >= NATIVE_MAP_VIEWPORT_COLUMNS || cursor_row >= NATIVE_MAP_VIEWPORT_ROWS {
        return Err(FdotherError::message(format!(
            "fdother: action overlay snapshot cursor ({cursor_column},{cursor_row}) is outside the {NATIVE_MAP_VIEWPORT_COLUMNS}x{NATIVE_MAP_VIEWPORT_ROWS} viewport"
        )));
    }
    Ok(NATIVE_ACTION_OVERLAY_BASE
        + NATIVE_ACTION_OVERLAY_STEP * (cursor_column - 1)
        + NATIVE_ACTION_OVERLAY_STEP * NATIVE_FRAMEBUFFER_STRIDE * (cursor_row - 1))
}

/// 還原先前擷取的原生 72×72 索引矩形。格式錯誤的備份在任何寫入前就拒收。
pub fn restore_action_overlay_snapshot(dst: &mut [u8], snapshot: &[u8], stride: i64, x: i64, y: i64) -> Result<()> {
    if stride <= 0 || dst.is_empty() || snapshot.len() as i64 != NATIVE_ACTION_OVERLAY_SNAPSHOT_BYTES {
        return Err(FdotherError::fixed("fdother: action overlay snapshot restore input is invalid"));
    }
    validate_action_overlay_snapshot_rect(dst, stride, x, y)?;
    let w = NATIVE_ACTION_OVERLAY_SNAPSHOT_WIDTH as usize;
    for row in 0..NATIVE_ACTION_OVERLAY_SNAPSHOT_HEIGHT {
        let at = row as usize * w;
        dst[snapshot_row(stride, x, y, row)].copy_from_slice(&snapshot[at..at + w]);
    }
    Ok(())
}

/// 實作 `0x1741c`／`0x179d5` 共用的幀緩衝位址式。`cursor_column`、`cursor_row` 是可見地圖游標座標；
/// 另外追蹤的鏡頭捲動全域不使用。
///
/// 這裡是可見游標 13×8 界線的**消費端**。原版不夾這個全域——走行捲動在鏡頭到邊界時照樣把它
/// 減到 -1（收據 docs/data/ui-traces/fd2-story-pan-cursor-20260909.json，frames idx=75）——但這條
/// 位址式把它當視窗內的格座標，出界就會寫到 0x8088 起算的視窗之外。出界時失敗即關閉，不猜原版在
/// 那種狀態會畫成什麼。
pub fn action_overlay_origin(cursor_column: i64, cursor_row: i64) -> Result<i64> {
    if cursor_column < 0 || cursor_row < 0 {
        return Err(FdotherError::fixed("fdother: negative action overlay origin"));
    }
    if cursor_column >= NATIVE_MAP_VIEWPORT_COLUMNS || cursor_row >= NATIVE_MAP_VIEWPORT_ROWS {
        return Err(FdotherError::message(format!(
            "fdother: action overlay cursor ({cursor_column},{cursor_row}) is outside the {NATIVE_MAP_VIEWPORT_COLUMNS}x{NATIVE_MAP_VIEWPORT_ROWS} viewport"
        )));
    }
    Ok(NATIVE_ACTION_OVERLAY_BASE
        + NATIVE_ACTION_OVERLAY_STEP * cursor_column
        + NATIVE_ACTION_OVERLAY_STEP * NATIVE_FRAMEBUFFER_STRIDE * cursor_row)
}

impl ActionOverlayState {
    /// 實作 FD2.EXE `0x1741c` 的表 ABI：
    ///
    /// ```text
    /// index = 3*firstArgumentWord + 2*secondArgumentWord
    /// ```
    ///
    /// 在 [`ActionOverlayState`] 中，firstArgumentWord 是 `direction_state`，secondArgumentWord 是
    /// `availability`。乘法順序刻意寫明，因為反過來會對戰鬥與 CONTINUE 呼叫端產生看似合理但錯誤的
    /// FDOTHER #2 格。
    ///
    /// 回傳的索引定址一個 FDOTHER #2 raw 格；不推論方向的可見圖示或可用性語意。乘加與 Go 一樣
    /// 以 64 位元繞回。
    pub fn cell_index(&self, direction: i64) -> Result<i64> {
        if direction < 0 || direction >= self.availability.len() as i64 {
            return Err(FdotherError::message(format!("fdother: action overlay direction {direction} is invalid")));
        }
        let availability = self.availability[direction as usize];
        let direction_state = self.direction_state[direction as usize];
        if availability < 0 || direction_state < 0 {
            return Err(FdotherError::message(format!(
                "fdother: negative action overlay state for direction {direction}"
            )));
        }
        Ok(3i64.wrapping_mul(direction_state).wrapping_add(2i64.wrapping_mul(availability)))
    }

    /// 重現 `sub_179D5` 於 `0x17A7D..0x17A8B` 的選中分支：穩態重繪對每個方向取 [`cell_index`]，只有
    /// 等於目前選擇的方向再加上閃爍相位。開合動畫由 `sub_1741C`／`sub_176B4` 擁有，兩者都沒有這個
    /// 加法，呼叫端不可套用。
    ///
    /// [`cell_index`]: Self::cell_index
    pub fn selected_cell_index(&self, direction: i64, blink_phase: i64) -> Result<i64> {
        let base = self.cell_index(direction)?;
        if !(0..=1).contains(&blink_phase) {
            return Err(FdotherError::message(format!(
                "fdother: action overlay blink phase {blink_phase} is outside the proven 0..1 range"
            )));
        }
        Ok(base.wrapping_add(blink_phase))
    }
}

/// 回傳原生 `0x1741c`／`0x176b4` 開啟或關閉動畫一幀所用的四個位元組偏移。它們是原生 stride 0x1c8
/// 幀緩衝內的偏移；呼叫端提供具體原點。這裡不暗示畫面錨點，因為它仍未證實。
pub fn action_overlay_frame_offsets(frame: i64, closing: bool) -> Result<[i64; 4]> {
    if !(0..4).contains(&frame) {
        return Err(FdotherError::message(format!("fdother: action overlay frame {frame} is invalid")));
    }
    let mut offsets = [0i64; 4];
    if closing {
        // 0x176b4 有獨立初始化的關閉序列；不是開啟幀倒過來。
        let start = [-0x23a0, 0x378, 0x3a8, 0x2ac0];
        let delta = [0x8e8, 6, -6, -0x8e8];
        for direction in 0..4 {
            offsets[direction] = start[direction] + (frame + 1) * delta[direction];
        }
        return Ok(offsets);
    }
    let delta = [-0x8e8, -6, 6, 0x8e8];
    for direction in 0..4 {
        offsets[direction] = 0x390 + (frame + 1) * delta[direction];
    }
    Ok(offsets)
}

/// 套用四個 raw FDOTHER #2 格的一幀原生動畫。`origin` 是位元組位址，而不是推論出的 x/y 錨點；
/// 偏移再以呼叫端的幀緩衝 stride 換算。
pub fn blit_action_overlay_frame(
    cells: &[RawCell],
    state: ActionOverlayState,
    dst: &mut [u8],
    stride: i64,
    origin: i64,
    frame: i64,
    closing: bool,
) -> Result<()> {
    if stride <= 0 || origin < 0 || origin >= dst.len() as i64 {
        return Err(FdotherError::fixed("fdother: action overlay origin is invalid"));
    }
    let offsets = action_overlay_frame_offsets(frame, closing)?;
    for (direction, offset) in offsets.into_iter().enumerate() {
        let index = state.cell_index(direction as i64)?;
        if index >= cells.len() as i64 {
            return Err(FdotherError::message(format!("fdother: action overlay cell {index} is absent")));
        }
        let pos = origin + offset;
        if pos < 0 || pos >= dst.len() as i64 {
            return Err(FdotherError::message(format!("fdother: action overlay position {pos} is invalid")));
        }
        cells[index as usize]
            .blit_at(dst, stride, pos % stride, pos / stride)
            .map_err(|e| FdotherError::message(format!("fdother: action overlay direction {direction}: {e}")))?;
    }
    Ok(())
}

/// `sub_17898` 於 `0x178CE 83 F8 03` 使用的界線：只有 BIOS 低字差值大於 3、或為負（`0x178D3`
/// `85 C0` 後 `7D` 的 jge 不成立）時才翻動閃爍相位。四個 tick 約 219.7 ms。
pub const ACTION_OVERLAY_BLINK_THRESHOLD: i64 = 3;

/// 保存 `sub_17898` 直接讀寫的兩個原始全域：`[0x53c13]` 的閃爍相位與 `[0x53c17]` 的上次翻動
/// 時間戳。兩者都是 BSS，初值為零；此處不另行植入起始值，以保留原版第一次比較的行為。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ActionOverlaySelectionBlink {
    /// `[0x53c13]`
    pub phase: i64,
    /// `[0x53c17]`
    pub tick: i64,
}

impl ActionOverlaySelectionBlink {
    /// 重現 `0x178BF..0x178F8`：以 `0x46C` 的 BIOS 低字與上次時間戳相減，差值超過界線或為負就把
    /// 相位在 0 與 1 之間翻動，並更新時間戳。
    ///
    /// Go 版接受 nil 接收者並直接返回；Rust 以 `&mut self` 排除 nil，該分支不存在。減法與相位遞增
    /// 照 Go 以 64 位元繞回；相位只有等於 2 才歸零，呼叫端塞入其他值時照樣遞增（同 Go）。
    pub fn advance(&mut self, raw_tick: i64) {
        let delta = raw_tick.wrapping_sub(self.tick);
        if (0..=ACTION_OVERLAY_BLINK_THRESHOLD).contains(&delta) {
            return;
        }
        self.phase = self.phase.wrapping_add(1);
        if self.phase == 2 {
            self.phase = 0;
        }
        self.tick = raw_tick;
    }
}

/// 重現 `sub_173E7`（`0x173F5..0x1741B`）：從 0 起逐一往上找第一個 availability 為零的方向。四個
/// 方向都不可用時回傳 4，與原版把 `[0x53c57]` 停在 4 的結果一致。
pub fn action_overlay_initial_direction(availability: [i64; 4]) -> i64 {
    availability.iter().position(|&a| a == 0).unwrap_or(availability.len()) as i64
}

/// 重現 `sub_177FC` 的四個方向分支（`0x17835..0x17897`）：掃描碼只有在該方向的 availability 為零時
/// 才會寫入 `[0x53c57]`；否則選擇不變，等待迴圈繼續。
pub fn action_overlay_accepts_direction(availability: [i64; 4], direction: i64) -> bool {
    (0..availability.len() as i64).contains(&direction) && availability[direction as usize] == 0
}
