//! 移植自 Go `spawn_intro.go`：`sub_32999` 的單位出場動畫。

use super::ch28_post_present::go_hex;
use super::separated_lmi1_bank::validate_native_spawn_intro_frames;
use super::{FdotherError, LMI1Entry, Result, decode_lmi1_resource};
use std::path::Path;

pub const NATIVE_SPAWN_INTRO_FRAME_RESOURCE: i64 = 9;
pub const NATIVE_SPAWN_INTRO_SOUND_RESOURCE: i64 = 95;
pub const NATIVE_SPAWN_INTRO_PASS_COUNT: i64 = 12;

/// `0x32A3D..0x32A6C` 的鏡頭閘門。和穩態單位層不同，Y 下界是 cameraY 而不是 cameraY-1。
/// 加減與 Go 一樣以 64 位元繞回。
pub fn native_spawn_intro_visible(
    x: i64,
    y: i64,
    camera_x: i64,
    camera_y: i64,
    visible_x_max: i64,
    visible_y_max: i64,
) -> bool {
    x >= camera_x.wrapping_sub(1)
        && x <= camera_x.wrapping_add(visible_x_max)
        && y >= camera_y
        && y <= camera_y.wrapping_add(visible_y_max).wrapping_add(1)
}

/// 重現 `0x32A6E..0x32AA9`。回傳的偏移相對於未改動的 `[0x53A49]` 工作緩衝基底；乘加與 Go 一樣
/// 以 64 位元繞回。
pub fn native_spawn_intro_lmi_origin(x: i64, y: i64, camera_x: i64, camera_y: i64) -> i64 {
    0x8088i64
        .wrapping_add(24i64.wrapping_mul(x.wrapping_sub(camera_x).wrapping_sub(1)))
        .wrapping_add((24 * 0x1c8i64).wrapping_mul(y.wrapping_sub(camera_y)))
        .wrapping_sub(0xab0)
}

/// 以原版 456 位元組 stride 套用 `0x4E85B` 的透明零 LMI 寫入。鏡頭裁切仍由呼叫端決定。
pub fn blit_native_spawn_intro_lmi(
    entry: &LMI1Entry,
    work: &mut [u8],
    x: i64,
    y: i64,
    camera_x: i64,
    camera_y: i64,
) -> Result<()> {
    let origin = native_spawn_intro_lmi_origin(x, y, camera_x, camera_y);
    if origin < 0 || origin >= work.len() as i64 {
        return Err(FdotherError::message(format!(
            "fdother: spawn-intro LMI origin {} outside work buffer",
            go_hex(origin)
        )));
    }
    entry.blit_at(work, 0x1c8, origin % 0x1c8, origin / 0x1c8, false)
}

/// 記錄 `sub_32999` 呈現之後的快照重建。語意標籤是附加的：原始 pass 索引與 raw 指標位移仍明列在
/// [`NativeSpawnIntroStep`]。
///
/// Go 版是 `uint8` 具名型別加 iota 常數，可以裝任意 0..255；這裡用同樣的新型別保留那個值域。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeSpawnIntroSnapshotMode(pub u8);

pub const NATIVE_SPAWN_INTRO_KEEP_SNAPSHOT: NativeSpawnIntroSnapshotMode = NativeSpawnIntroSnapshotMode(0);
pub const NATIVE_SPAWN_INTRO_SPLIT_UNITS: NativeSpawnIntroSnapshotMode = NativeSpawnIntroSnapshotMode(1);
pub const NATIVE_SPAWN_INTRO_FULL_FRAME: NativeSpawnIntroSnapshotMode = NativeSpawnIntroSnapshotMode(2);

/// `0x32999` 的一次迴圈。呈現發生在 `snapshot_mode` 之前。`sound_resource` 除了 pass 1 都是 -1；
/// pass 1 把 raw FDOTHER #95 酬載交給 `0x25A96`，不推論音訊語意。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeSpawnIntroStep {
    pub pass: i64,
    pub lmi_entry: i64,
    pub sound_resource: i64,
    pub delay_ticks: i64,
    pub snapshot_mode: NativeSpawnIntroSnapshotMode,
    pub redraw_terrain: bool,
    pub new_unit_pixel_lift: i64,
}

/// 保存全部十二次迴圈與三次呈現後的快照重建。負的 lift 對應暫時的 `[0x53A49]` 指標減法：
/// -0xE40/456=-8 與 -0x8E8/456=-5 列。
pub fn native_spawn_intro_schedule() -> Vec<NativeSpawnIntroStep> {
    let mut steps: Vec<NativeSpawnIntroStep> = (0..NATIVE_SPAWN_INTRO_PASS_COUNT)
        .map(|pass| NativeSpawnIntroStep {
            pass,
            lmi_entry: pass,
            sound_resource: -1,
            delay_ticks: 1,
            ..Default::default()
        })
        .collect();
    steps[1].sound_resource = NATIVE_SPAWN_INTRO_SOUND_RESOURCE;
    steps[6].snapshot_mode = NATIVE_SPAWN_INTRO_SPLIT_UNITS;
    steps[6].new_unit_pixel_lift = -8;
    steps[7].snapshot_mode = NATIVE_SPAWN_INTRO_SPLIT_UNITS;
    steps[7].redraw_terrain = true;
    steps[7].new_unit_pixel_lift = -5;
    steps[8].snapshot_mode = NATIVE_SPAWN_INTRO_FULL_FRAME;
    steps[8].redraw_terrain = true;
    steps
}

pub fn validate_native_spawn_intro_schedule(steps: &[NativeSpawnIntroStep]) -> Result<()> {
    let want = native_spawn_intro_schedule();
    if steps.len() != want.len() {
        return Err(FdotherError::message(format!(
            "fdother: spawn-intro schedule has {} passes, want {}",
            steps.len(),
            want.len()
        )));
    }
    for (i, (got, want)) in steps.iter().zip(&want).enumerate() {
        if got != want {
            return Err(FdotherError::message(format!("fdother: spawn-intro pass {i} differs from 0x32999")));
        }
    }
    Ok(())
}

/// 讀 FDOTHER #9，拒收任何不符合 `0x32999` 消費的十二個項目幾何的資源。封存雜湊／版本閘門仍歸
/// 專案參考清單。
pub fn decode_native_spawn_intro_frames(dat_path: impl AsRef<Path>) -> Result<Vec<LMI1Entry>> {
    let entries = decode_lmi1_resource(dat_path, NATIVE_SPAWN_INTRO_FRAME_RESOURCE)?;
    validate_native_spawn_intro_frames(&entries)?;
    Ok(entries)
}
