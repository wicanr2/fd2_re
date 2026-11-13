//! Go `native_command_rng_walk_test.go`。

use super::native_command4_test::command4_test_animation;
use super::*;
use crate::native_rng::native_rng_step;

// Go: TestWalkNativeCommand4RNGMatchesR4Receipt
// 第六章 r4 收據 seq 1451：敵方 18 對 (10,17) 施指令 4，0x4E893 入口序列是六次 0x26A5A（mode 0）、
// 0x1C7F2、0x1C86E、然後 0x2AF45／0x26BCD 交錯，入口亂數 28590、下一個 0x13A9F 入口 27109。
#[test]
fn walk_native_command4_rng_matches_r4_receipt() {
    let schedule = build_native_command4_presentation_schedule(0, Some(&command4_test_animation())).unwrap();
    let mut resolved_at = 0u16;
    let mut resolve = |_: i64, rng: u16| -> Resolved {
        resolved_at = rng;
        Ok((native_rng_step(native_rng_step(rng)), true))
    };
    let last = walk_native_command5_rng(28590, schedule, 0, 1, Some(&mut resolve)).unwrap();
    // 六步 mode 0 之後才擲命中：28590→…→44735；命中 63129（29<85）、傷害 13676（76）。
    assert_eq!(resolved_at, 44735, "resolve rng");
    assert_eq!(last, 27109, "final rng");
    // 走過畫格序列的 RNG 要和純亂數 walk 一致（同一條序列）。
    let mut state = new_native_command5_state_for_schedule(28590, schedule);
    for _ in 0..schedule.front_frames {
        state = plan_native_command5_draw_frame(state, schedule, 0).unwrap().next;
    }
    state.rng = native_rng_step(native_rng_step(state.rng));
    let (_, after) = build_native_command5_target_sequence(state, schedule, 0).unwrap();
    let (_, tail) = build_native_command5_tail_sequence(after, schedule, 0).unwrap();
    assert_eq!(tail.rng, 27109, "sequence rng");
}

// Go: TestWalkNativeCommand0RNGMatchesR4Receipt
// 第六章 r4 收據 seq 114：敵方 19 對 (10,18) 施指令 0，序列是 0x1C7F2、0x1C86E、七次 0x2AF45；
// 入口亂數 19233、下一個 0x13A9F 入口 13566。
#[test]
fn walk_native_command0_rng_matches_r4_receipt() {
    let schedule =
        NativeCommand0PresentationSchedule { frames: NATIVE_COMMAND0_PRESENTATION_FRAMES, ..Default::default() };
    let mut resolved_at = 0u16;
    let mut resolve = |_: i64, rng: u16| -> Resolved {
        resolved_at = rng;
        Ok((native_rng_step(native_rng_step(rng)), true))
    };
    let last = walk_native_command0_rng(19233, schedule, 1, Some(&mut resolve)).unwrap();
    assert!(resolved_at == 19233 && last == 13566, "resolve rng={resolved_at} final={last} want 19233／13566");
    let markers = (0..NATIVE_COMMAND0_PRESENTATION_FRAMES).filter(|&step| native_command0_marker_step(step)).count();
    assert_eq!(markers as i64, NATIVE_COMMAND0_DAMAGE_STAGES, "marker steps");
}

// Go: TestWalkNativeCommand0RNGMissConsumesOnlyTheHitRoll
// 第六章 r4 收據 seq 1013：敵方 19 的指令 0 未命中（22346→15095），只吃 0x1C7F2 一步，沒有 0x2AF45 抖動。
#[test]
fn walk_native_command0_rng_miss_consumes_only_the_hit_roll() {
    let schedule =
        NativeCommand0PresentationSchedule { frames: NATIVE_COMMAND0_PRESENTATION_FRAMES, ..Default::default() };
    let mut resolve = |_: i64, rng: u16| -> Resolved { Ok((native_rng_step(rng), false)) };
    assert_eq!(walk_native_command0_rng(22346, schedule, 1, Some(&mut resolve)).unwrap(), 15095, "final rng");
}

// Go: TestWalkNativeCommand4RNGMissSkipsMarkerShake
// 指令 4 未命中：handler 的六槽重置照樣吃亂數，但沒有 HP 分段與抖動。
#[test]
fn walk_native_command4_rng_miss_skips_marker_shake() {
    let schedule = build_native_command4_presentation_schedule(0, Some(&command4_test_animation())).unwrap();
    let mut resolve = |_: i64, rng: u16| -> Resolved { Ok((native_rng_step(rng), false)) };
    let last = walk_native_command5_rng(28590, schedule, 0, 1, Some(&mut resolve)).unwrap();
    // 6 步初始化 + 1 步命中 + 4 次重置 = 11 步；命中版是 18 步。
    let want = (0..11).fold(28590u16, |rng, _| native_rng_step(rng));
    assert_eq!(last, want, "final rng");
}
