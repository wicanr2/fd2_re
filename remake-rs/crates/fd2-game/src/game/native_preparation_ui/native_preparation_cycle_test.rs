//! Go 來源：`remake/cmd/fd2/native_preparation_cycle_test.go`（6 個測試，全部移植）。受測的
//! `stepNativePreparationCycleTick`／`stepNativePreparationUILifecycle` 在 `native_preparation_ui.go`，
//! 後兩個測試經 `stepNativeClassUILifecycle`（`native_class_ui_lifecycle.go`）。
//!
//! Go `time.Unix(s, 0)` 對應 `GameInstant(s × 10⁹)`；`nativeBIOSTickPeriod` 是 `NATIVE_BIOS_TICK_PERIOD`。

use super::*;

fn unix(s: i64) -> GameInstant {
    GameInstant(s * 1_000_000_000)
}

fn ticks(n: i64) -> GoDuration {
    GoDuration(n * NATIVE_BIOS_TICK_PERIOD.0)
}

fn preparation_campaign() -> campaign::Runner {
    let c = campaign::Campaign {
        start: String::from("prep"),
        nodes: BTreeMap::from([(
            String::from("prep"),
            campaign::Node { r#type: String::from("preparation"), ..Default::default() },
        )]),
        ..Default::default()
    };
    campaign::new_runner(&c)
}

/// Go：`native_preparation_cycle_test.go:11` `TestNativePreparationCycleUses1297DBoundariesAndVisibleSequence`
#[test]
fn test_native_preparation_cycle_uses_1297d_boundaries_and_visible_sequence() {
    let mut g = Game::default();
    for (tick, state, visible) in [(0, 0, 0), (4, 0, 0), (5, 1, 1), (9, 1, 1), (10, 2, 2), (15, 3, 1), (20, 0, 0)] {
        g.step_native_preparation_cycle_tick(tick);
        let got = fdicon::native_frame_index(0, false, g.prep_idle_cycle, 0);
        assert!(
            g.prep_idle_cycle == state && got.as_ref().is_ok_and(|v| *v == visible),
            "tick={tick} state={} visible={got:?}, want state={state} visible={visible}",
            g.prep_idle_cycle
        );
    }
}

/// Go：`native_preparation_cycle_test.go:38` `TestNativePreparationCycleAdvancesOnSignedBIOSWrap`
#[test]
fn test_native_preparation_cycle_advances_on_signed_bios_wrap() {
    let mut g = Game { prep_idle_cycle: 2, prep_last_tick: 0x7fff, ..Default::default() };
    g.step_native_preparation_cycle_tick(-0x8000);
    assert!(
        g.prep_idle_cycle == 3 && g.prep_last_tick == -0x8000,
        "wrapped state={} last={}",
        g.prep_idle_cycle,
        g.prep_last_tick
    );
}

/// Go：`native_preparation_cycle_test.go:46` `TestNativePreparationLifecycleRunsOnlyForActiveSelection`
#[test]
fn test_native_preparation_lifecycle_runs_only_for_active_selection() {
    let mut g = Game {
        camp: Some(preparation_campaign()),
        native_preparation_ui: Some(NativePreparationUIAssets::default()),
        ..Default::default()
    };
    let start = unix(1);
    g.step_native_preparation_ui_lifecycle(start);
    g.step_native_preparation_ui_lifecycle(start + ticks(5));
    assert_eq!(g.prep_idle_cycle, 0, "inactive preparation advanced");
    g.prep_selecting = true;
    g.step_native_preparation_ui_lifecycle(start + ticks(10));
    g.step_native_preparation_ui_lifecycle(start + ticks(15));
    assert!(
        g.prep_idle_cycle == 1 && g.prep_last_tick == 5,
        "active preparation state={} last={}",
        g.prep_idle_cycle,
        g.prep_last_tick
    );
    g.camp.as_mut().unwrap().cur = String::from("missing");
    g.step_native_preparation_ui_lifecycle(start + ticks(20));
    assert_eq!(g.prep_idle_cycle, 1, "non-preparation node advanced");
}

/// Go：`native_preparation_cycle_test.go:75` `TestNativePreparationFixedHashShotFreezesRequestedCycle`
#[test]
fn test_native_preparation_fixed_hash_shot_freezes_requested_cycle() {
    let mut g = Game {
        camp: Some(preparation_campaign()),
        native_preparation_ui: Some(NativePreparationUIAssets::default()),
        prep_selecting: true,
        prep_idle_cycle: 2,
        prep_shot_cycle_frozen: true,
        ..Default::default()
    };
    g.step_native_preparation_ui_lifecycle(unix(100));
    g.step_native_preparation_ui_lifecycle(unix(200));
    assert_eq!(g.prep_idle_cycle, 2, "固定雜湊截圖相位被時鐘推進");
}

/// Go：`native_preparation_cycle_test.go:95` `TestNativePreparationConfirmationUses19953PulseCadence`
#[test]
fn test_native_preparation_confirmation_uses_19953_pulse_cadence() {
    let mut g = Game { camp: Some(preparation_campaign()), prep_confirm: true, ..Default::default() };
    let start = unix(10);
    g.step_native_class_ui_lifecycle(start);
    g.step_native_class_ui_lifecycle(start + ticks(1));
    assert_eq!(g.native_class_ui_pulse, 0, "one BIOS tick advanced confirmation pulse");
    g.step_native_class_ui_lifecycle(start + ticks(2));
    assert_eq!(g.native_class_ui_pulse, 1, "two BIOS ticks left confirmation pulse");
    g.prep_confirm = false;
    g.prep_selecting = true;
    g.step_native_class_ui_lifecycle(start + ticks(4));
    assert_eq!(g.native_class_ui_pulse, 1, "inactive preparation confirmation advanced pulse");
}

/// Go：`native_preparation_cycle_test.go:123` `TestNativePreparationPromptUses19953PulseCadence`
#[test]
fn test_native_preparation_prompt_uses_19953_pulse_cadence() {
    let mut g = Game { camp: Some(preparation_campaign()), ..Default::default() };
    let start = unix(20);
    g.step_native_class_ui_lifecycle(start);
    g.step_native_class_ui_lifecycle(start + ticks(2));
    assert_eq!(g.native_class_ui_pulse, 1, "preparation prompt pulse");
    g.prep_selecting = true;
    g.step_native_class_ui_lifecycle(start + ticks(4));
    assert_eq!(g.native_class_ui_pulse, 1, "inactive prompt advanced pulse");
}
