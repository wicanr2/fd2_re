//! 呈現回報（`present.rs`）的單元測試：Go `Draw` 各分支的前置條件與失敗路徑。

use super::*;

fn map_game() -> Game {
    Game { m: Some(MapData::default()), st: Some(battle::State::default()), ..Default::default() }
}

fn vga() -> Vec<u8> {
    vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize]
}

#[test]
fn begin_present_frame_clears_frame_flags() {
    // Go main.go:9356..9368：每幀開頭清本幀呈現旗標。
    let mut g = Game { native_map_presented: true, native_interface_hud_visible: true, ..map_game() };
    g.modern_hd_world_visible = true;
    g.begin_present_frame();
    assert!(!g.native_map_presented && !g.native_interface_hud_visible && !g.modern_hd_world_visible);
}

#[test]
fn palette_ramp_acknowledged_only_with_complete_frame() {
    // Go native_palette_ramp.go:164：vga 與 256 色調色盤齊全才設 drawn；不齊時 Draw 只印除錯字。
    let mut g = map_game();
    g.native_palette_ramp =
        Some(NativePaletteRampJob { vga: vga(), palette: vec![ColorRgba::default(); 256], ..Default::default() });
    assert_eq!(g.acknowledge_present(), PresentOwner::NativePaletteRamp);
    assert!(g.native_palette_ramp.as_ref().unwrap().drawn);

    let mut g = map_game();
    g.native_palette_ramp =
        Some(NativePaletteRampJob { vga: vga(), palette: vec![ColorRgba::default(); 255], ..Default::default() });
    assert_eq!(g.acknowledge_present(), PresentOwner::NativePaletteRamp);
    let job = g.native_palette_ramp.as_ref().expect("工作留著");
    assert!(!job.drawn);
}

#[test]
fn end_turn_recovery_without_dac_is_cancelled() {
    // Go main.go:9765..9768：drawNativeEndTurnRecovery 回 false 時 cancelNativeEndTurnRecovery。
    let mut g = map_game();
    g.native_end_turn_recovery = Some(NativeEndTurnRecoveryJob { frames: [vga(), vga(), vga()], ..Default::default() });
    assert_eq!(g.acknowledge_present(), PresentOwner::NativeEndTurnRecovery);
    assert!(g.native_end_turn_recovery.is_none());
    assert!(g.load_err.contains("native END recovery: Draw unavailable"), "{}", g.load_err);

    let mut g = map_game();
    g.native_map_dac = vec![0; 768];
    g.native_end_turn_recovery = Some(NativeEndTurnRecoveryJob { frames: [vga(), vga(), vga()], ..Default::default() });
    g.acknowledge_present();
    assert!(g.native_end_turn_recovery.as_ref().unwrap().drawn);
    assert!(g.load_err.is_empty());
}

#[test]
fn deferred_turn_staging_flash_is_acknowledged_on_every_path() {
    // Go main.go:9386..9388：非 indexed 的白閃在 Draw 的 defer 裡設 drawn，連 startupBlocked 也一樣。
    let mut g = Game { startup_blocked: true, ..map_game() };
    g.native_turn_staging =
        Some(NativeTurnStagingJob { phase: NATIVE_TURN_STAGING_FLASH, indexed: false, ..Default::default() });
    assert_eq!(g.acknowledge_present(), PresentOwner::StartupBlocked);
    assert!(g.native_turn_staging.as_ref().unwrap().drawn);
}

#[test]
fn growth_roll_observer_reaches_battle_state() {
    // Go main.go:4087：bindCommandLearn 把 growthRollObserver 交給 st.NativeGrowthRollObserver。
    let g = Game {
        growth_roll_observer: Some(battle::Callback::new(Arc::new(|_: &battle::Unit, _: u16| 7u16))),
        ..Default::default()
    };
    let mut st = battle::State::default();
    g.bind_command_learn(Some(&mut st));
    let observer = st.native_growth_roll_observer.expect("observer");
    assert_eq!((observer.0)(&battle::Unit::default(), 1), 7);
}
