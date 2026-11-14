//! Go `native_level_up_dialogue_test.go` 與 `native_transient_phase_test.go:311`（`nativeMessagePages`）。

use super::*;

/// `battle::Unit`／`battle::State`／`fdother::Frame` 有私有欄位，不能用結構更新語法；先 `default()` 再一次寫入欄位。
macro_rules! build {
    ($t:ty { $($f:ident : $v:expr),* $(,)? }) => {{
        let mut x = <$t>::default();
        ($(x.$f,)*) = ($($v,)*);
        x
    }};
}

/// Go `fakeLevelUpStrings`：假的 FDTXT_000，只放 0x1E292 會用到的幾句。
pub(crate) struct FakeLevelUpStrings(pub(crate) BTreeMap<i64, Vec<u16>>);

impl NativeMessageStrings for FakeLevelUpStrings {
    fn words(&self, index: i64) -> Result<Vec<u16>> {
        self.0.get(&index).cloned().ok_or_else(|| game_err!("no string {}", battle::go_sharp_x(index)))
    }
}

/// Go `levelUpFixtureStrings`：控制碼照原版（0x1E8 只有 FFFA；0x1E9 尾端 FFFD；0x1EA..0x1EE FFFA＋FFFD；
/// 0x24B FFFC；0x1B9+id 是指令名）。
fn level_up_fixture_strings() -> FakeLevelUpStrings {
    FakeLevelUpStrings(BTreeMap::from([
        (0x1e8, vec![100, 0xfffa, 101]),
        (0x1e9, vec![102, 0xfffd]),
        (0x1ea, vec![110, 0xfffa, 111, 0xfffd]),
        (0x1eb, vec![112, 0xfffa, 111, 0xfffd]),
        (0x1ec, vec![113, 0xfffa, 111, 0xfffd]),
        (0x1ed, vec![114, 0xfffa, 111, 0xfffd]),
        (0x1ee, vec![115, 0xfffa, 111, 0xfffd]),
        (0x24b, vec![120, 0xfffc, 121]),
        (0x1b9 + 5, vec![130, 131]),
    ]))
}

/// Go：`native_level_up_dialogue_test.go:36`
#[test]
fn test_native_level_up_dialogue_lines_split_pages_at_fffd_and_skip_zero_growth() {
    let pending = PendingNativeLevelUp {
        exp: 16,
        ups: vec![battle::LevelUpEvent {
            new_lv: 5,
            ap_gain: 7,
            dp_gain: 4,
            dx_gain: 0,
            hp_gain: 10,
            mp_gain: 0,
            learned_command_ids: vec![5],
        }],
        ..Default::default()
    };
    let (pages, last_wait) = native_level_up_dialogue_lines(&level_up_fixture_strings(), &pending).unwrap();
    // r4 seq 968..972：第一頁「得到經驗16點！／等級上升了！！」，之後每欄一頁；速度與 MMP 為零不寫；
    // 「學會了」在最後一欄的 FFFD 之後寫、本身沒有 FFFD，所以是最後一頁且不等鍵。
    let want: Vec<Vec<Vec<u16>>> = vec![
        vec![vec![100, 1, 6, 101], vec![102]],
        vec![vec![110, 7, 111]],
        vec![vec![112, 4, 111]],
        vec![vec![114, 1, 0, 111]],
        vec![vec![120, 130, 131, 121]],
    ];
    assert_eq!(pages, want);
    assert!(!last_wait);
    let pending = PendingNativeLevelUp {
        exp: 16,
        ups: vec![battle::LevelUpEvent { new_lv: 5, ap_gain: 7, ..Default::default() }],
        ..Default::default()
    };
    let (pages, last_wait) = native_level_up_dialogue_lines(&level_up_fixture_strings(), &pending).unwrap();
    assert_eq!(pages.len(), 2, "最後一欄以 FFFD 收尾要等鍵：{pages:?}");
    assert!(last_wait);
}

/// Go：`native_level_up_dialogue_test.go:63`
#[test]
fn test_native_level_up_dialogue_lines_without_level_up_is_one_unwaited_page() {
    let pending = PendingNativeLevelUp { exp: 9, ..Default::default() };
    let (pages, last_wait) = native_level_up_dialogue_lines(&level_up_fixture_strings(), &pending).unwrap();
    assert_eq!(pages, vec![vec![vec![100u16, 9, 101]]]);
    assert!(!last_wait);
}

/// Go：`native_level_up_dialogue_test.go:73`。Go 的單位不在 `g.st` 裡；Rust 以 `st.units` 的索引參照。
#[test]
fn test_queue_native_level_up_dialogue_follows_raw_experience_consumer() {
    let mut g = Game::default();
    let enemy = build!(battle::Unit { has_native_record_byte6: true, native_record_byte6: 0 });
    let player = build!(battle::Unit { has_native_record_byte6: true, native_record_byte6: 2 });
    g.st = Some(build!(battle::State { units: vec![enemy, player] }));
    g.queue_native_level_up_dialogue(Some(0), 12, &[]);
    g.queue_native_level_up_dialogue(Some(1), 0, &[]);
    g.queue_native_level_up_dialogue(Some(1), 12, &[]);
    assert_eq!(g.pending_native_level_ups.len(), 2, "{:?}", g.pending_native_level_ups);
    assert_eq!(g.pending_native_level_ups[0].unit, Some(0));
    assert_eq!(g.pending_native_level_ups[1].unit, Some(1));
    assert_eq!(g.pending_native_level_ups[1].exp, 12);
}

/// D 組補充：沒有 raw +6 的單位不排對話（Go 的 `!unit.HasNativeRecordByte6` 早退）。
#[test]
fn queue_native_level_up_dialogue_skips_units_without_raw_side() {
    let mut g = Game { st: Some(build!(battle::State { units: vec![battle::Unit::default()] })), ..Default::default() };
    g.queue_native_level_up_dialogue(Some(0), 12, &[]);
    g.queue_native_level_up_dialogue(None, 12, &[]);
    assert!(g.pending_native_level_ups.is_empty());
}

/// Go：`native_transient_phase_test.go:311`。0x1A8C3 把 MaxHP/10 存進 [0x53AE1]，0x15F84 把 FFFA 寫成十進位數字、
/// FFFE 換行。
#[test]
fn test_native_message_pages_splits_damage_text_lines() {
    let strings = FakeLevelUpStrings(BTreeMap::from([(0x1e7, vec![90, 0xfffe, 91, 0xfffa, 92])]));
    let (pages, wait) = native_message_pages(&strings, 0x1e7, 12).unwrap();
    assert!(!wait);
    assert_eq!(pages, vec![vec![vec![90u16], vec![91, 1, 2, 92]]]);
}

/// D 組補充：`nativeMessagePages` 與 `expandNativeMessageWords` 的錯誤字串與 Go 逐字相同（`%#x`）。
#[test]
fn native_message_words_errors_match_go_text() {
    let strings =
        FakeLevelUpStrings(BTreeMap::from([(0x1e7, vec![0xfffc]), (0x1e8, vec![1, 0xfffe]), (0x1e9, vec![])]));
    assert_eq!(
        native_message_pages(&strings, 0x1e7, 0).unwrap_err().message(),
        "message text 0x1e7 needs a name argument"
    );
    assert_eq!(native_message_pages(&strings, 0x1e9, 0).unwrap_err().message(), "message text 0x1e9 is empty");
    assert_eq!(
        expand_native_message_words(&strings, 0x1e8, 0, &[]).unwrap_err().message(),
        "message text 0x1e8 has an unexpected line break"
    );
}
