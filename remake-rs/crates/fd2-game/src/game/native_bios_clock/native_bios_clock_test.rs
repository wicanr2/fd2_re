//! Go 來源：`remake/cmd/fd2/native_bios_clock_test.go`。
//!
//! Go 測試用的 `Seed` 是 `r8` 函式（第一章沒有執行到，骨架沒有產生）；這裡照它的本體
//! （`native_bios_clock.go:27`）直接寫入欄位，期望值不改。

use super::*;

fn unix(sec: i64) -> GameInstant {
    GameInstant(sec * GoDuration::SECOND.0)
}

fn after(t: GameInstant, d: i64) -> GameInstant {
    t + GoDuration(d)
}

/// Go `nativeBIOSClock.Seed` 的本體（驗證範圍後寫入 `last` 與 `elapsedTicks`）。
fn seed(clock: &mut NativeBIOSClock, raw_tick: i64, now: GameInstant) -> bool {
    if now.is_zero() || !(-0x8000..=0x7fff).contains(&raw_tick) {
        return false;
    }
    *clock = NativeBIOSClock { last: now, elapsed_ticks: raw_tick as i16 as u16 as u64, ..Default::default() };
    true
}

fn materialized_native_clock_game() -> Game {
    let mut st = battle::State::default();
    (st.w, st.h) = (13, 8);
    st.append_native_map_selector_batch(Vec::new()).unwrap();
    st.materialize_native_map_view_state(battle::NativeMapViewState::default()).unwrap();
    Game { st: Some(st), ..Default::default() }
}

const P: i64 = NATIVE_BIOS_TICK_PERIOD.0;

/// Go：`native_bios_clock_test.go:22` `TestNativeBIOSClockUsesMonotonicBattleLocalLowWord`
#[test]
fn test_native_bios_clock_uses_monotonic_battle_local_low_word() {
    let mut clock = NativeBIOSClock::default();
    let start = unix(100);
    assert_eq!(clock.sample(start), 0, "initial tick");
    assert_eq!(clock.sample(after(start, 5 * P + P / 2)), 5, "five ticks");
    assert_eq!(clock.sample(after(start, 5 * P)), 5, "backward wall sample changed monotonic tick");
    clock.elapsed_ticks = 0x7fff;
    clock.remainder = GoDuration(0);
    clock.last = start;
    assert_eq!(clock.sample(after(start, P)), -0x8000, "signed low-word wrap");
}

/// Go：`native_bios_clock_test.go:41` `TestNativeBIOSClockSeedsCapturedSignedLowWord`
#[test]
fn test_native_bios_clock_seeds_captured_signed_low_word() {
    let mut clock = NativeBIOSClock::default();
    let start = unix(150);
    assert!(seed(&mut clock, -123, start), "valid signed title tick was rejected");
    assert_eq!(clock.sample(start), -123, "seed sample");
    assert_eq!(clock.sample(after(start, 5 * P)), -118, "elapsed seeded sample");
    let before = clock;
    assert!(!seed(&mut clock, 0x8000, start) && clock == before, "invalid seed changed clock");
}

/// Go：`native_bios_clock_test.go:59` `TestNativeBIOSClockCurrentDoesNotAdvance`
#[test]
fn test_native_bios_clock_current_does_not_advance() {
    let mut clock = NativeBIOSClock::default();
    assert!(!clock.current().1, "unseeded clock exposed a current tick");
    let now = unix(900);
    assert!(seed(&mut clock, 123, now), "clock seed rejected");
    let before = clock;
    assert_eq!(clock.current(), (123, true));
    assert_eq!(clock, before, "Current advanced or mutated the clock");
}

/// Go：`native_bios_clock_test.go:77` `TestGameAdvancesAllNativeMapFrameTimingFromOneBIOSSample`
#[test]
fn test_game_advances_all_native_map_frame_timing_from_one_bios_sample() {
    let mut g = materialized_native_clock_game();
    let start = unix(200);
    assert!(g.advance_native_map_clock(start), "initial native map clock sample rejected");
    assert_eq!(g.st.as_ref().unwrap().native_map_cycle_state.moving, 1, "moving cycle after first 0x1297d call");
    assert!(g.advance_native_map_clock(after(start, 5 * P)), "five-tick native map clock sample rejected");
    let st = g.st.as_ref().unwrap();
    let c = &st.native_map_cycle_state;
    assert!(c.idle == 1 && c.moving == 2 && c.last_timer_tick == 5, "sprite cycles={c:?}");
    let p = &st.native_terrain_phase_state;
    assert!(p.phase == 1 && p.last_timer_tick == 5, "terrain phase={p:?}");
    let f = &st.native_terrain_flip_state;
    assert!(f.value == 1 && f.last_timer_tick == 5, "terrain flip={f:?}");
    let u = &st.native_unit_pixel_shift_state;
    assert!(u.value == 1 && u.last_timer_tick == 5, "unit pixel shift={u:?}");
}

/// Go：`native_bios_clock_test.go:104` `TestGameNativeMapClockFailsClosedWithoutView`
#[test]
fn test_game_native_map_clock_fails_closed_without_view() {
    let mut g = materialized_native_clock_game();
    g.st.as_mut().unwrap().has_native_map_view_state = false;
    let before = g.st.as_ref().unwrap().native_map_cycle_state;
    assert!(
        !g.advance_native_map_clock(unix(300)) && g.st.as_ref().unwrap().native_map_cycle_state == before,
        "partial native runtime advanced"
    );
}
