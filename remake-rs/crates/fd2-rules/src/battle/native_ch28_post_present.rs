//! Go 來源：`remake/internal/battle/native_ch28_post_present.go`（153 行）。

use super::*;

pub const NATIVE_CH28_POST_VISIBLE_COLUMNS: i64 = 13;

pub const NATIVE_CH28_POST_VISIBLE_ROWS: i64 = 8;

/// NativeCh28PostOverlayTarget is one eligible record and the exact work-buffer
/// destination saved by 0x1DB65 before any pose or inactive-byte writes.
///
/// Go：`native_ch28_post_present.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCh28PostOverlayTarget {
    pub slot: i64,
    pub origin: i64,
}

/// NativeCh28PostPresentationPlan preserves the raw viewport and the stable
/// record-pointer set used by both six-frame LMI loops. It deliberately carries
/// no inferred effect name.
///
/// Go：`native_ch28_post_present.go:25`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCh28PostPresentationPlan {
    pub camera_x: i64,
    pub camera_y: i64,
    pub targets: Vec<NativeCh28PostOverlayTarget>,
}

/// Go：`native_ch28_post_present.go:30`，`func validateNativeCh28PostPresenterFields(st *State) error`
pub(crate) fn validate_native_ch28_post_presenter_fields(st: Option<&State>) -> Result<()> {
    let Some(st) = st.filter(|s| !s.units.is_empty()) else {
        return Err(battle_err!("ch28 post presenter: battle units unavailable"));
    };
    for (index, unit) in st.units.iter().enumerate() {
        if !unit.has_native_map_presentation || !unit.has_native_record_byte5 || unit.hp < 0 || unit.hp > 0xffff {
            return Err(battle_err!("ch28 post presenter: slot{index} lacks consumed raw fields"));
        }
    }
    if !st.has_native_runtime_unit_projection {
        if !st.native_runtime_records.is_empty() {
            return Err(battle_err!("ch28 post presenter: partial raw projection"));
        }
        return Ok(());
    }
    if st.native_runtime_records.len() != st.units.len() {
        return Err(battle_err!("ch28 post presenter: inconsistent raw projection"));
    }
    for (index, unit) in st.units.iter().enumerate() {
        let record = &st.native_runtime_records[index].raw;
        let raw = &unit.native_map_presentation;
        if record[0] != raw.x
            || record[1] != raw.y
            || record[3] != raw.pose
            || record[5] != unit.native_record_byte5
            || u16::from_le_bytes([record[0x40], record[0x41]]) != unit.hp as u16
        {
            return Err(battle_err!("ch28 post presenter: slot{index} raw projection disagrees with typed fields"));
        }
    }
    Ok(())
}

/// PlanNativeCh28PostPresentation reproduces 0x1DB7C..0x1DC2D. The native
/// upper bounds are cameraX+13 and cameraY+8+1; the lower bounds are camera-1.
/// Empty Targets is valid and selects the original no-presentation branch.
///
/// Go：`native_ch28_post_present.go:64`，`func PlanNativeCh28PostPresentation(st *State, cameraX, cameraY int) (NativeCh28PostPresentationPlan, error)`
pub fn plan_native_ch28_post_presentation(
    st: Option<&State>,
    camera_x: i64,
    camera_y: i64,
) -> Result<NativeCh28PostPresentationPlan> {
    validate_native_ch28_post_presenter_fields(st)?;
    let Some(st) = st else { unreachable!("驗證已拒收 nil 狀態") };
    let mut plan = NativeCh28PostPresentationPlan { camera_x, camera_y, targets: Vec::new() };
    for (index, unit) in st.units.iter().enumerate() {
        if unit.native_record_byte5 & 1 != 0 || unit.hp != 0 {
            continue;
        }
        let raw = &unit.native_map_presentation;
        let (x, y) = (raw.x as i64, raw.y as i64);
        if x < camera_x - 1
            || x > camera_x + NATIVE_CH28_POST_VISIBLE_COLUMNS
            || y < camera_y - 1
            || y > camera_y + NATIVE_CH28_POST_VISIBLE_ROWS + 1
        {
            continue;
        }
        let origin = fdother::native_ch28_post_overlay_origin(x, y, camera_x, camera_y)
            .map_err(|e| battle_err!("ch28 post presenter: slot{index}: {e}"))?;
        plan.targets.push(NativeCh28PostOverlayTarget { slot: index as i64, origin });
    }
    Ok(plan)
}

/// ApplyNativeCh28PostPoseFrame reproduces the only record write in each of
/// 0x1DB65's thirteen first-phase frames: eligible active records with word40=0
/// receive raw +3 = frame mod 4. The renderer/present/tick boundary is owned by
/// the caller and must succeed before advancing to the next frame.
///
/// Go：`native_ch28_post_present.go:92`，`func ApplyNativeCh28PostPoseFrame(st *State, frame int) error`
pub fn apply_native_ch28_post_pose_frame(st: Option<&mut State>, frame: i64) -> Result<()> {
    if !(0..fdother::NATIVE_CH28_POST_POSE_FRAMES).contains(&frame) {
        return Err(battle_err!("ch28 post presenter: invalid pose frame {frame}"));
    }
    validate_native_ch28_post_presenter_fields(st.as_deref())?;
    let Some(st) = st else { unreachable!("驗證已拒收 nil 狀態") };
    // 驗證已保證有投影時 `native_runtime_records` 與 `units` 等長；Go 在改寫前重查一次，結果相同。
    let pose = (frame % 4) as u8;
    for (index, unit) in st.units.iter_mut().enumerate() {
        if unit.native_record_byte5 & 1 != 0 || unit.hp != 0 {
            continue;
        }
        unit.native_map_presentation.pose = pose;
        unit.dir = frame % 4;
        if st.has_native_runtime_unit_projection {
            st.native_runtime_records[index].raw[3] = pose;
        }
    }
    Ok(())
}

/// ApplyNativeCh28PostInactiveMark reproduces 0x1DD25..0x1DD50. Every record
/// whose raw word40 is zero receives raw +5=1, including off-viewport records.
///
/// Go：`native_ch28_post_present.go:127`，`func ApplyNativeCh28PostInactiveMark(st *State) error`
pub fn apply_native_ch28_post_inactive_mark(st: Option<&mut State>) -> Result<()> {
    validate_native_ch28_post_presenter_fields(st.as_deref())?;
    let Some(st) = st else { unreachable!("驗證已拒收 nil 狀態") };
    for (index, unit) in st.units.iter_mut().enumerate() {
        if unit.hp == 0 {
            unit.native_record_byte5 = 1;
            if st.has_native_runtime_unit_projection {
                st.native_runtime_records[index].raw[5] = 1;
            }
        }
    }
    Ok(())
}

#[cfg(test)]
mod native_ch28_post_present_test;
