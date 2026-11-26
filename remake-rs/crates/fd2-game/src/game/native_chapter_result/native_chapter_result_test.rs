//! Go 來源：`remake/cmd/fd2/native_chapter_result_test.go`（期望值不改）。
//!
//! 未移植：`:45`、`:101`、`:157`（第 13 章需逐幀推進對白的測具，章重播 `ch13-sample-r2` 另驗）。

use super::*;

/// Go：`native_chapter_result_test.go:114` `TestChapter15SilentResultHandsOffToNativeDefeat`。
/// 直接進章與注入 raw 測試條件只驗 E1；普通輸入由章重播另驗。
#[test]
fn test_chapter15_silent_result_hands_off_to_native_defeat() {
    if f_child_env(module_path!(), "test_chapter15_silent_result_hands_off_to_native_defeat", |dir| {
        vec![
            ("FD2_TITLE", "0".into()),
            ("FD2_MUTE", "1".into()),
            ("FD2_SEED", "4".into()),
            ("FD2_CAMPAIGN", CANONICAL_CAMPAIGN_REFERENCE.into()),
            ("FD2_CAMP_NODE", "battle_ch15".into()),
            ("FD2_NATIVE_SAVE", dir.join("FD2.SAV").into_os_string()),
            ("XDG_DATA_HOME", dir.as_os_str().to_os_string()),
        ]
    })
    .is_none()
    {
        return;
    }
    let mut g = load_game().expect("loadGame");
    assert!(g.load_err.is_empty(), "{}", g.load_err);
    // Go 的原生地圖時鐘讀 time.Now；Rust 讀虛擬時鐘，沒有跑過 Update 時是零值，給一個非零時刻。
    g.virtual_now = GameInstant(1_000_000_000);
    let rules = &g.sc.as_ref().expect("第 15 章劇本").native_result_rules;
    assert!(rules.len() == 1 && rules[0].actions.is_empty(), "canonical正式入口未載入無對白結果規則");
    g.st.as_mut().unwrap().units[65].native_record_byte5 |= 1;
    g.check_result();
    assert!(
        g.result.is_empty() && g.native_defeat.is_none() && g.load_err.is_empty(),
        "其他友軍倒下被判敗：{}",
        g.load_err
    );
    // 無對白列不產生對白 owner；敗北提示沿用 FDOTHER79。
    g.st.as_mut().unwrap().units[64].native_record_byte5 |= 1;
    g.check_result();
    assert!(
        g.result == "lose"
            && g.native_defeat.is_some()
            && g.battle_event.is_none()
            && g.native_chapter_result.is_none()
            && g.load_err.is_empty()
            && g.native_result_matched_rules == ["record64_inactive"],
        "無對白結果沒有直接交接原生敗北：result={:?} error={}",
        g.result,
        g.load_err
    );
    // 純結果 owner 沒有對白素材也能判定；正式敗北 owner 仍要求戰場／色盤。
    (g.native_chapter_result, g.native_defeat, g.battle_event) = (None, None, None);
    (g.result, g.load_err) = (String::new(), String::new());
    (g.camp, g.native_map_assets) = (None, None);
    g.begin_native_chapter_result();
    assert!(
        g.load_err.is_empty() && g.result == "lose" && g.battle_event.is_none(),
        "無對白結果列錯誤要求戰場對白素材：{}",
        g.load_err
    );
}
