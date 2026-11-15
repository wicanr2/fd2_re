//! C 組：END 回收與 AI 閒置回收（0x13FD4）的測試。Go 來源：`native_end_turn_recovery_test.go`、
//! `native_ai_idle_recovery_test.go`。
//!
//! Go 夾具 `endRecoveryTestGame` 以 `t.Setenv("FD2_MUTE", "1")` 關掉 sample4；工作區禁止 `unsafe`，
//! Rust 改在子行程設這個變數（F 組的 `f_child_env`）。夾具先合成一次原生整幀，`composeNativeMapFrame`
//! 經過 B 組的 `nativeMapHUDInput`，所以這些測試先標「等 B 合併」。
//! 需要 `Draw`／`Update` 的 Go 測試列在 PORTING.md「C 組 R5 測試清單」。

use super::*;
use crate::game::native_map_frame_input::native_map_frame_input_test::complete_native_map_frame_fixture;
use std::ffi::OsString;

fn mute_env(_: &std::path::Path) -> Vec<(&'static str, OsString)> {
    vec![("FD2_MUTE", OsString::from("1"))]
}

/// Go：`native_end_turn_recovery_test.go:13` `endRecoveryTestGame`（`FD2_MUTE` 由呼叫端的子行程設定）。
fn end_recovery_test_game() -> Game {
    let (a, m, mut st) = complete_native_map_frame_fixture();
    let u = &mut st.units[0];
    (u.on_field, u.hp, u.max_hp, u.native_record_byte6) = (true, 5, 10, 2);
    u.set_map_placement(4, 4, 0);
    let mut g = Game { native_map_assets: Some(a), m: Some(m), st: Some(st), cur_x: 4, cur_y: 4, ..Default::default() };
    g.compose_native_map_frame_at_with_palette_cycle(GameInstant(100_000_000_000), false, true).expect("compose");
    g
}

fn unit0(g: &mut Game) -> &mut battle::Unit {
    &mut g.st.as_mut().unwrap().units[0]
}

/// Go：`native_end_turn_recovery_test.go:46` `TestNativeEndTurnRecoveryAutoENDUsesSameOwner`
#[test]
fn test_native_end_turn_recovery_auto_end_uses_same_owner() {
    if f_child_env(module_path!(), "test_native_end_turn_recovery_auto_end_uses_same_owner", mute_env).is_none() {
        return;
    }
    let mut g = end_recovery_test_game();
    let u = unit0(&mut g);
    (u.camp, u.acted, u.native_record_byte5) = (battle::OWN, true, 0x80);
    assert!(
        g.auto_end_player_phase(Some(0)) && g.native_end_turn_recovery.is_some() && g.load_err.is_empty(),
        "自動 END 沒有進同一 owner：{}",
        g.load_err
    );
    let job_plan_len = g.native_end_turn_recovery.as_ref().unwrap().plan.len();
    g.end_turn();
    assert!(
        g.native_end_turn_recovery.is_some() && job_plan_len == 0 && unit0(&mut g).hp == 5 && !g.ai_busy,
        "重入或全員已行動的 END 提前寫入／啟動 AI"
    );
}

/// Go：`native_end_turn_recovery_test.go:133` `TestNativeEndTurnRecoveryUsesOpaqueMaskAndRawRestore`
#[test]
fn test_native_end_turn_recovery_uses_opaque_mask_and_raw_restore() {
    if f_child_env(module_path!(), "test_native_end_turn_recovery_uses_opaque_mask_and_raw_restore", mute_env).is_none()
    {
        return;
    }
    let mut g = end_recovery_test_game();
    // Every frame of the fixture has opaque pixels. Make source0 opaque and
    // the neighboring pixel transparent, then compose the formal source.
    for sprite in g.native_map_assets.as_mut().unwrap().units.as_mut().unwrap().sprites.iter_mut() {
        (sprite.pixels[0], sprite.mask[1]) = (0, 0);
    }
    g.compose_native_map_frame_at_with_palette_cycle(GameInstant(100_000_000_000), false, true).expect("compose");
    let before = g.native_map_vga.clone();
    g.begin_native_end_turn_recovery(None).expect("begin");
    let job = g.native_end_turn_recovery.as_ref().unwrap();
    const PIXEL: usize = (4 * 24 - 2) * 320 + 4 + 4 * 24; // VGA y4 plus native -6 row bias
    assert!(
        job.frames[1][PIXEL] == 0xc8
            && job.frames[1][PIXEL + 1] == before[PIXEL + 1]
            && job.frames[2][PIXEL] == 0
            && job.frames[2] == before,
        "mode2/source0/透明/raw restore：{:x} {:x} {:x}",
        job.frames[1][PIXEL],
        job.frames[1][PIXEL + 1],
        job.frames[2][PIXEL]
    );
}

/// Go：`native_end_turn_recovery_test.go:156` `TestNativeEndTurnRecoveryRejectsAssetsAndStalePlanWithoutWrites`
#[test]
fn test_native_end_turn_recovery_rejects_assets_and_stale_plan_without_writes() {
    if f_child_env(
        module_path!(),
        "test_native_end_turn_recovery_rejects_assets_and_stale_plan_without_writes",
        mute_env,
    )
    .is_none()
    {
        return;
    }
    for kind in ["asset", "hp", "selector", "gate"] {
        let mut g = end_recovery_test_game();
        let before_gate = g.st.as_ref().unwrap().native_map_hud_state.display_gate_b;
        if kind == "asset" {
            g.native_map_assets.as_mut().unwrap().units.as_mut().unwrap().sprites = Vec::new();
            assert!(g.begin_native_end_turn_recovery(None).is_err(), "{kind}: 缺資產未拒絕");
        } else {
            g.begin_native_end_turn_recovery(None).expect("begin");
            let job = g.native_end_turn_recovery.as_mut().unwrap();
            (job.frame, job.drawn) = (1, true); // isolates stale transaction preflight
            let ready_at = job.ready_at;
            let u = unit0(&mut g);
            match kind {
                "hp" => u.max_hp = 11, // same floor(/5); must still reject
                "selector" => u.native_map_presentation.pose = 1,
                _ => u.native_record_byte6 = 0,
            }
            g.step_native_end_turn_recovery(ready_at);
            assert!(!g.load_err.is_empty(), "{kind}: 過期交易或sample未拒絕");
        }
        let gate = g.st.as_ref().unwrap().native_map_hud_state.display_gate_b;
        let u = unit0(&mut g);
        assert!(
            u.hp == 5 && u.native_record_byte5 == 0 && gate == before_gate && g.native_end_turn_recovery.is_none(),
            "{kind}: 失敗交易留下HP/flags/HUD寫入"
        );
    }
}

/// Go：`native_end_turn_recovery_test.go:193` `TestNativeEndTurnRecoveryOffscreenAndNoCandidate`
#[test]
fn test_native_end_turn_recovery_offscreen_and_no_candidate() {
    if f_child_env(module_path!(), "test_native_end_turn_recovery_offscreen_and_no_candidate", mute_env).is_none() {
        return;
    }
    for acted in [false, true] {
        let mut g = end_recovery_test_game();
        let u = unit0(&mut g);
        u.set_map_placement(30, 30, 0);
        if acted {
            u.native_record_byte5 = 0x80;
        }
        let before = g.native_map_vga.clone();
        g.begin_native_end_turn_recovery(None).expect("begin");
        let job = g.native_end_turn_recovery.as_mut().unwrap();
        assert!(job.frames[1] == before && job.frames[2] == before, "畫面外候選仍寫pixel");
        (job.frame, job.drawn) = (1, true);
        let ready_at = job.ready_at;
        g.step_native_end_turn_recovery(ready_at);
        let cue = g.native_end_turn_recovery.as_ref().map_or(-1, |job| job.cue_count);
        let hp = unit0(&mut g).hp;
        assert!(
            (!acted && hp == 7 && cue == 1) || (acted && hp == 5 && cue == 0),
            "畫面外／無候選的數值或cue錯誤：acted={acted} hp={hp} cue={cue}"
        );
    }
}

/// Go：`native_ai_idle_recovery_test.go:14` `aiIdleRecoveryTestGame`。
fn ai_idle_recovery_test_game() -> (Game, battle::NativeAIIdleRecoveryDecision) {
    let mut g = end_recovery_test_game();
    let u = unit0(&mut g);
    (u.native_record_byte6, u.camp) = (0, battle::ENEMY);
    (u.has_native_record_byte34, u.has_native_record_byte35, u.has_native_record_byte36) = (true, true, true);
    u.has_native_record_word46 = true;
    (u.inventory_slots, u.native_inventory_flags) = (vec![0; 8].into(), vec![0; 8].into());
    u.has_native_record_byte8 = true;
    let st = g.st.as_ref().unwrap();
    let records = battle::native_ai_scoring_records(&st.units).expect("records");
    let decision = battle::plan_native_ai_idle_recovery(&records, st.units.len() as i64, 0).expect("plan");
    assert!(decision.accepted, "AI 回復測試來源 {decision:?}");
    (g, decision)
}

/// Go：`native_ai_idle_recovery_test.go:34` `TestNativeAIIdleRecoveryInitialMaskAndRawRestore`
#[test]
fn test_native_ai_idle_recovery_initial_mask_and_raw_restore() {
    if f_child_env(module_path!(), "test_native_ai_idle_recovery_initial_mask_and_raw_restore", mute_env).is_none() {
        return;
    }
    let (mut g, decision) = ai_idle_recovery_test_game();
    for sprite in g.native_map_assets.as_mut().unwrap().units.as_mut().unwrap().sprites.iter_mut() {
        (sprite.pixels[0], sprite.mask[1]) = (0, 0);
    }
    unit0(&mut g).native_record_byte5 = 0x80; // mode0 must bypass the steady-frame gray LUT.
    g.compose_native_map_frame_at_with_palette_cycle(GameInstant(100_000_000_000), false, true).expect("compose");
    let st = g.st.as_ref().unwrap();
    let (entry, _) = st.units[0].native_unit_layer_entry();
    let view = st.native_map_view_state;
    let offset = fdicon::native_placement_offset(
        entry.x,
        entry.y,
        view.camera_x,
        view.camera_y,
        entry.pose,
        entry.motion_offset,
        0,
        false,
    )
    .expect("offset") as usize;
    const PIXEL: usize = (4 * 24 - 2) * 320 + 4 + 4 * 24;
    // A later layer in the initial work snapshot must not replace raw mode0.
    (g.native_map_work[offset + 2], g.native_map_vga[PIXEL + 2]) = (0x44, 0x44);
    let before = g.native_map_vga.clone();
    let presentation = battle::build_native_ai_idle_recovery_presentation(decision).expect("presentation");
    let frames = g.build_native_ai_idle_recovery_frames(Some(0), &presentation).expect("frames");
    assert_eq!(frames.len(), 3);
    assert!(frames[0] == before, "第一次 wait 應呈現初始畫面");
    assert!(
        frames[1][PIXEL] == 0xc8 && frames[1][PIXEL + 1] == before[PIXEL + 1] && frames[1][PIXEL + 2] == 0xc8,
        "mode2 未保留 source0 寫入或透明 span"
    );
    assert!(
        frames[2][PIXEL] == 0 && frames[2][PIXEL + 2] == 3,
        "mode0 必須恢復 raw 像素，不用已行動 LUT 或 composer 快照"
    );
}

/// Go：`native_ai_idle_recovery_test.go:101` `TestNativeAIIdleRecoveryMissingAssetsAndInvalidTupleLeaveHP`
#[test]
fn test_native_ai_idle_recovery_missing_assets_and_invalid_tuple_leave_hp() {
    if f_child_env(module_path!(), "test_native_ai_idle_recovery_missing_assets_and_invalid_tuple_leave_hp", mute_env)
        .is_none()
    {
        return;
    }
    for kind in ["asset", "tuple"] {
        let (mut g, decision) = ai_idle_recovery_test_game();
        let before_range = g.st.as_ref().unwrap().native_map_range_mode;
        if kind == "asset" {
            g.native_map_assets.as_mut().unwrap().units.as_mut().unwrap().sprites = Vec::new();
            assert!(g.begin_native_ai_idle_recovery(Some(0), decision, None).is_err(), "缺素材仍開始回復");
        } else {
            let mut presentation = battle::build_native_ai_idle_recovery_presentation(decision).expect("presentation");
            presentation.first_decode.mode = 0;
            assert!(g.build_native_ai_idle_recovery_frames(Some(0), &presentation).is_err(), "非法 tuple 仍產生畫面");
        }
        let range = g.st.as_ref().unwrap().native_map_range_mode;
        let u = unit0(&mut g);
        assert!(
            u.hp == 5 && !u.acted && g.native_ai_idle_recovery.is_none() && range == before_range,
            "{kind}: 失敗留下 HP／行動／range 交易"
        );
    }
}

/// Go：`native_ai_idle_recovery_test.go:126` `TestNativeAIIdleRecoveryViewportCopiesVerified312x192Region`
#[test]
fn test_native_ai_idle_recovery_viewport_copies_verified312x192_region() {
    let mut work = vec![0u8; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
    let vga = vec![0u8; indexedmap::NATIVE_MAP_VGA_SIZE as usize];
    for y in 0..NATIVE_AI_IDLE_RECOVERY_HEIGHT {
        work[(NATIVE_AI_IDLE_RECOVERY_WORK_BASE + y * NATIVE_AI_IDLE_RECOVERY_STRIDE) as usize] = (y + 1) as u8;
    }
    let got = native_ai_idle_recovery_viewport(&work, &vga).expect("viewport");
    for y in 0..NATIVE_AI_IDLE_RECOVERY_HEIGHT {
        let at = (NATIVE_AI_IDLE_RECOVERY_VGA_OFFSET + y * NATIVE_AI_IDLE_RECOVERY_VGA_STRIDE) as usize;
        assert_eq!(got[at], (y + 1) as u8, "row {y}");
    }
    assert_eq!(
        got[(NATIVE_AI_IDLE_RECOVERY_VGA_OFFSET + NATIVE_AI_IDLE_RECOVERY_WIDTH - 1) as usize],
        0,
        "copy unexpectedly wrote beyond 312-byte row"
    );
}

/// Go：`native_ai_idle_recovery_test.go:147` `TestNativeAIIdleRecoveryViewportRejectsShortBuffersBeforeWrite`
#[test]
fn test_native_ai_idle_recovery_viewport_rejects_short_buffers_before_write() {
    let work = vec![0u8; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize - 1];
    let vga = vec![0u8; indexedmap::NATIVE_MAP_VGA_SIZE as usize];
    assert!(native_ai_idle_recovery_viewport(&work, &vga).is_err(), "short native work buffer unexpectedly accepted");
}

/// Rust 新增：`unitIndex` 對應 Go 以指標相等在 `st.Units` 找單位；`osMuteOrShot` 在沒有 `Game` 時為真。
#[test]
fn unit_index_and_mute_gate() {
    let mut st = battle::State::default();
    st.units = vec![battle::Unit::default(); 2];
    assert_eq!(unit_index(Some(&st), Some(1)), 1);
    assert_eq!(unit_index(Some(&st), Some(2)), -1);
    assert_eq!(unit_index(None, Some(0)), -1);
    assert_eq!(unit_index(Some(&st), None), -1);
    assert!(os_mute_or_shot(None));
    let g = Game { shot_path: String::from("x.png"), ..Default::default() };
    assert!(os_mute_or_shot(Some(&g)));
}
