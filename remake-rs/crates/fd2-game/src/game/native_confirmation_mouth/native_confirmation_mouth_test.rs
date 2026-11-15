//! R5 階段 2 r5d：`composeNativeConfirmationMouth` 與全域亂數的測試。
//!
//! Go `native_confirmation_mouth_test.go` 的 3 個測試經 `newPreparationRecordTestGame`（整備素材夾具）、
//! `composeNativePreparationPromptFrame`（城鎮與設施組，`r5`）與 `beginNativeSystemEndTurn`（crate 裡還沒有）驅動，
//! 本組不移植，留給主題組（見 `PORTING.md`「R5 階段 2：r5d」）。這裡直接測本組函式的分支與 Go 全域亂數的取用順序。

use super::*;

fn portrait(fill: u8) -> dato::Frame {
    dato::Frame { width: 80, height: 80, pixels: vec![fill; 80 * 80] }
}

fn end_turn_owner_game() -> Game {
    Game {
        native_system_end_turn_confirm: true,
        native_system_end_turn_ui: Some(Box::default()),
        native_confirmation_mouth_owner: Some(NATIVE_CONFIRMATION_OWNER_END_TURN),
        ..Default::default()
    }
}

/// Rust 新增：沒有等待中的擁有者、擁有者不同或嘴巴閉著時原樣回傳，`ok` 只看長度（Go `len(frame) == 320*200`）。
#[test]
fn test_compose_native_confirmation_mouth_passes_frame_without_open_owner() {
    let frame: Vec<u8> = (0..320 * 200).map(|i| (i % 251) as u8).collect();
    let portraits = vec![portrait(1), portrait(2), portrait(3), portrait(4)];
    let g = Game::default();
    assert_eq!(g.compose_native_confirmation_mouth(&frame, &portraits), (frame.clone(), true), "沒有擁有者");
    assert_eq!(
        g.compose_native_confirmation_mouth(&frame[..10], &portraits),
        (frame[..10].to_vec(), false),
        "長度不對"
    );
    let mut g = end_turn_owner_game();
    assert_eq!(g.compose_native_confirmation_mouth(&frame, &portraits), (frame.clone(), true), "嘴巴閉著");
    g.native_confirmation_mouth.open = true;
    g.native_confirmation_mouth_owner = Some(OwnerToken(7));
    assert_eq!(g.compose_native_confirmation_mouth(&frame, &portraits), (frame.clone(), true), "擁有者不同");
}

/// Rust 新增：等待中的擁有者張嘴時以 `portraits[3]` 合成（Go `ComposeNativeConfirmationPortrait`），缺第 3 幀或來源
/// 不完整時失敗；合成不改呼叫端的來源與嘴型狀態。
#[test]
fn test_compose_native_confirmation_mouth_uses_caller_frame3() {
    let frame: Vec<u8> = (0..320 * 200).map(|i| (i % 249) as u8).collect();
    let portraits = vec![portrait(1), portrait(2), portrait(3), portrait(0x44)];
    let mut g = end_turn_owner_game();
    g.native_confirmation_mouth.open = true;
    let before = g.native_confirmation_mouth;
    let want = campaign::compose_native_confirmation_portrait(&frame, &portraits[3]).expect("合成");
    assert_ne!(want, frame, "測試夾具的頭像沒有改到畫面");
    assert_eq!(g.compose_native_confirmation_mouth(&frame, &portraits), (want, true));
    assert_eq!(g.native_confirmation_mouth, before, "繪圖推進了嘴型狀態");
    assert_eq!(g.compose_native_confirmation_mouth(&frame, &portraits[..3]), (Vec::new(), false), "缺第 3 幀");
    assert_eq!(g.compose_native_confirmation_mouth(&frame[..100], &portraits), (Vec::new(), false), "來源不完整");
}

/// Rust 新增：`go_global_rand` 有值時，故事對白嘴型的 `rand.Intn(30)` 與確認框嘴型的
/// `rand.New(rand.NewSource(rand.Int63()))` 依取用順序共用 Go 全域序列（`rand.Seed(4)`），R4 的故事嘴型固定種子來源不建立。
#[test]
fn test_go_global_rand_is_shared_in_go_call_order() {
    let mut global = battle::GoMathRand::new(4);
    let mut g = Game { go_global_rand: Some(battle::GoMathRand::new(4)), ..Default::default() };
    let mut want = Vec::new();
    let mut got = Vec::new();
    for _ in 0..5 {
        want.push(global.intn(30));
        got.push(g.story_mouth_intn30());
    }
    // 確認框嘴型第一次取樣：以全域 Int63 當種子建立私有來源。
    let mut private = battle::GoMathRand::new(global.int63());
    for _ in 0..2 {
        want.push(private.intn(30));
        got.push(g.sample_native_confirmation_mouth_random());
    }
    // 之後的故事嘴型接著取全域序列；確認框不再碰全域來源。
    for _ in 0..5 {
        want.push(global.intn(30));
        got.push(g.story_mouth_intn30());
    }
    assert_eq!(got, want);
    assert!(g.story_mouth_rng.is_none(), "全域亂數打開時不得建立 R4 的故事嘴型固定種子來源");
}

/// Rust 新增：`go_global_rand` 為 `None`（擷取關閉）時照舊用 R4 的兩個固定種子。
#[test]
fn test_mouth_rng_keeps_r4_fixed_seeds_without_global_rand() {
    let mut g = Game::default();
    let a = g.story_mouth_intn30();
    let b = g.sample_native_confirmation_mouth_random();
    assert_eq!(a, battle::GoMathRand::new(STORY_MOUTH_RNG_SEED).intn(30));
    assert_eq!(b, battle::GoMathRand::new(CONFIRMATION_MOUTH_RNG_SEED).intn(30));
    assert!(g.go_global_rand.is_none());
}
