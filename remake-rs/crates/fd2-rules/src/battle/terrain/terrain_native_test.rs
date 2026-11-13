//! 移植自 `remake/internal/battle/terrain_native_test.go`。

use super::*;

/// Go：`TestNativeTerrainAPDPPct`
#[test]
fn native_terrain_apdp_pct_table() {
    let cases = [
        (0u8, 5, 0, true),
        (1, 0, 0, true),
        (2, -5, 10, true),
        (3, -5, 10, true),
        (4, -5, -5, true),
        (5, 0, 0, true),
        (6, 0, 0, false),
    ];
    for (code, ap, dp, accepted) in cases {
        assert_eq!(native_terrain_apdp_pct(code), (ap, dp, accepted), "code {code}");
    }
}

/// Go：`TestTerrainAPDPPct_PrefersValidatedNativeCellCode`
#[test]
fn terrain_apdp_pct_prefers_validated_native_cell_code() {
    let st = State {
        w: 3,
        h: 2,
        cost: Some(vec![1, 1, 1, 1, 1, 1]),
        native_terrain_move_codes: vec![0, 1, 2, 3, 4, 5],
        ..Default::default()
    };
    let want = [(5, 0), (0, 0), (-5, 10), (-5, 10), (-5, -5), (0, 0)];
    for (i, pair) in want.into_iter().enumerate() {
        let i = i as i64;
        assert_eq!(st.terrain_apdp_pct(i % st.w, i / st.w), pair, "cell {i}");
    }
}

/// Go：`TestTerrainAPDPPct_IncompleteNativeCodesUseLegacyFallback`
#[test]
fn terrain_apdp_pct_incomplete_native_codes_use_legacy_fallback() {
    let st = State { w: 2, h: 1, cost: Some(vec![1, 2]), native_terrain_move_codes: vec![0], ..Default::default() };
    assert_eq!(st.terrain_apdp_pct(0, 0), (5, -5), "legacy flat fallback");
    assert_eq!(st.terrain_apdp_pct(1, 0), (-5, -5), "legacy swamp fallback");
}
