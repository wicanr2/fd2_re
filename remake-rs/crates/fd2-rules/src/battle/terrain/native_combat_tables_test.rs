//! 移植自 `remake/internal/battle/native_combat_tables_test.go`。

use super::*;
use crate::test_support::docs_file;

/// Go `loadNativeCombatTables`：docs/data/exe_tables/native_combat_tables.json 的最小視圖，
/// 回傳 (class_crit_pct, terrain_ap_pct, terrain_dp_pct) 的 `values`。缺檔時回 `None`。
fn load_native_combat_tables() -> Option<(Vec<i64>, Vec<i64>, Vec<i64>)> {
    let path = docs_file("data/exe_tables/native_combat_tables.json")?;
    let v: serde_json::Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    let values = |key: &str| -> Vec<i64> {
        v["tables"][key]["values"].as_array().unwrap().iter().map(|x| x.as_i64().unwrap()).collect()
    };
    Some((values("class_crit_pct"), values("terrain_ap_pct"), values("terrain_dp_pct")))
}

/// Go：`TestNativeTerrainAPDPPctMatchesTheExeTables` 釘住 NativeTerrainAPDPPct 就是原版
/// 的 0x51A12／0x51A2A 兩張表。它們各只有 6 格，第 7 格起就是 FDTXT.DAT 的字串，
/// 所以界外一定要失敗即關閉，不能沿用最後一格。
#[test]
fn native_terrain_apdp_pct_matches_the_exe_tables() {
    let Some((_, ap, dp)) = load_native_combat_tables() else { return };
    assert!(ap.len() == 6 && dp.len() == 6, "表長 ap={} dp={}，原版各 6 格", ap.len(), dp.len());
    for code in 0..6u8 {
        let (got_ap, got_dp, ok) = native_terrain_apdp_pct(code);
        assert!(ok, "地形碼 {code} 被判成不認得");
        let c = code as usize;
        assert!(
            got_ap == ap[c] && got_dp == dp[c],
            "地形碼 {code} 是 ({got_ap},{got_dp})，原版是 ({},{})",
            ap[c],
            dp[c]
        );
    }
    assert!(!native_terrain_apdp_pct(6).2, "地形碼 6 在原版表外，必須失敗即關閉");
}

/// Go：`TestClassCritPctMatchesTheExeTable` 釘住每個職業的暴擊率就是 0x5239B。
/// 這個值原版是 sub_29F72 的暴擊擲骰門檻（還要加上武器 record +10 的加成，
/// 那一段重製端尚未接，見 worklist remake-attack-missing-counterattack）。
#[test]
fn class_crit_pct_matches_the_exe_table() {
    let Some((crit, _, _)) = load_native_combat_tables() else { return };
    assert!(crit.len() >= 24, "暴擊表只有 {} 格", crit.len());
    let Some(path) = docs_file("data/exe_tables/resist_crit.json") else { return };
    let rows: serde_json::Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    let rows = rows.as_array().unwrap();
    assert!(!rows.is_empty(), "resist_crit.json 是空的");
    for row in rows {
        // Go 解碼成 struct：缺欄位或型別不符的欄位是零值。
        let class = row["cls"].as_i64().unwrap_or(0);
        let name = row["name"].as_str().unwrap_or("");
        let crit_pct = row["crit_pct"].as_i64().unwrap_or(0);
        let index = class - 1;
        assert!(index >= 0 && (index as usize) < crit.len(), "職業 {class}（{name}）落在 0x5239B 之外");
        assert_eq!(crit[index as usize], crit_pct, "職業 {class}（{name}）暴擊率，0x5239B[{index}]");
    }
}
