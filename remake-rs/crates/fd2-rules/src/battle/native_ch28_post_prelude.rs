//! Go 來源：`remake/internal/battle/native_ch28_post_prelude.go`（136 行）。

use super::*;

/// Go：`native_ch28_post_prelude.go:8`
pub(crate) static NATIVE_CH28_POST_FRONTIERS: LazyLock<BTreeSet<i64>> =
    LazyLock::new(|| BTreeSet::from([76, 78, 80, 82, 84, 86, 87, 88, 90, 93]));

/// ValidateNativeCh28PostFrontier accepts only the materialization prefixes
/// produced by the recovered map28 chain: 20 persistent records, group8's 56
/// rows, groups4..7 in two-row order, then group1's three rows. Event72 may
/// append groups2/3 atomically between complete groups. Group9 stays pending.
/// READY: docs/data/ida/fd2_ch29_flow_20261017.json.
///
/// Go：`native_ch28_post_prelude.go:17`，`func ValidateNativeCh28PostFrontier(st *State) error`
pub fn validate_native_ch28_post_frontier(st: Option<&State>) -> Result<()> {
    todo!("battle/native_ch28_post_prelude.go:ValidateNativeCh28PostFrontier")
}

/// ApplyNativeCh28PostRawPrelude reproduces only the proven raw writers before
/// sub_1DB65: clear word +0x40 from slot20 through the current tail, then set
/// slot20 raw +7/+8 to 0x7e. The 0x1DB65 indexed presenter is a separate gate;
/// callers must not treat this transaction as the complete postbattle beat.
///
/// Go：`native_ch28_post_prelude.go:82`，`func ApplyNativeCh28PostRawPrelude(st *State) error`
pub fn apply_native_ch28_post_raw_prelude(st: Option<&mut State>) -> Result<()> {
    todo!("battle/native_ch28_post_prelude.go:ApplyNativeCh28PostRawPrelude")
}
