//! Go `remake/cmd/fd2/chapter_parity_legacy_end_test.go`：舊 oracle 輸出缺 `before_seq` 時，從原版 checkpoint
//! 補回 END 的系統選單來源（#179）。第一章的 END 都帶 `before_seq`，補回路徑在 R8 移植。

use super::*;

/// Go `chapter_parity_legacy_end_test.go:22`：`recoverParityLegacyENDSources`。回傳 `None` 對應 Go 的 nil 切片
/// （`replay-settings.json` 寫 `null`）。
pub(crate) fn recover_parity_legacy_end_sources(actions: &[ParityAction]) -> Option<Vec<serde_json::Value>> {
    for pair in actions.windows(2) {
        let (clear, end) = (&pair[0], &pair[1]);
        if clear.kind != "force_enemy_clear" || end.kind != "end_turn" || end.before_seq.is_some() {
            continue;
        }
        todo!("R8: chapter_parity_legacy_end_test.go:49 舊 oracle 的 END before_seq 補回")
    }
    None
}
