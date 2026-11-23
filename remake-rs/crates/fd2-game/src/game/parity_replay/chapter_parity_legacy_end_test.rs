//! Go `remake/cmd/fd2/chapter_parity_legacy_end_test.go`：舊 oracle 輸出缺 `before_seq` 時，從原版 checkpoint
//! 補回 END 的系統選單來源（#179）。第一章的 END 都帶 `before_seq`，補回路徑在 R8 移植。

use super::*;
use std::path::Path;

#[derive(serde::Deserialize)]
struct OwnerCheckpoint {
    exe_sha256: String,
    input_chain: Vec<String>,
    view: CursorView,
}

#[derive(serde::Deserialize)]
struct CursorView {
    round: i64,
    cursor_x: i64,
    cursor_y: i64,
}

impl OwnerCheckpoint {
    fn has(&self, address: &str) -> bool {
        self.input_chain.iter().any(|entry| entry == address)
    }
}

fn read_owner(run: &Path, seq: i64) -> Result<(Vec<u8>, OwnerCheckpoint)> {
    let path = run.join(format!("checkpoint-{seq:04}.json"));
    let raw = std::fs::read(&path).map_err(|err| game_err!("{}：{err}", path.display()))?;
    let checkpoint: OwnerCheckpoint =
        serde_json::from_slice(&raw).map_err(|err| game_err!("{}：{err}", path.display()))?;
    if checkpoint.exe_sha256 != super::chapter_parity_start_test::PARITY_EXE_SHA256 {
        return Err(game_err!("END 檢查點 {seq} 的原版執行檔雜湊不符"));
    }
    Ok((raw, checkpoint))
}

/// #179：只用相鄰的原版游標與系統選單收據補回 END 的 before_seq。
pub(crate) fn recover_parity_legacy_end_sources(
    run: &Path,
    actions: &mut [ParityAction],
    keys: Option<&BTreeMap<i64, String>>,
) -> Result<Option<Vec<serde_json::Value>>> {
    let mut sources = Vec::new();
    for index in 1..actions.len() {
        let (previous, remaining) = actions.split_at_mut(index);
        let (clear, end) = (&previous[index - 1], &mut remaining[0]);
        if clear.kind != "force_enemy_clear" || end.kind != "end_turn" || end.before_seq.is_some() {
            continue;
        }
        if clear.round != end.round || clear.seq >= end.seq || end.at.len() != 2 {
            return Err(game_err!("清場與 END 檢查點的順序或回合不符"));
        }
        let (_, start) = read_owner(run, clear.seq)?;
        if start.view.round != clear.round || !start.has("0x117F8") {
            return Err(game_err!("清場檢查點缺少原版游標入口"));
        }
        let keys = keys.ok_or_else(|| game_err!("缺少 END 的原版按鍵紀錄"))?;
        let mut proven = None;
        for (&seq, key) in keys.range((clear.seq + 1)..end.seq) {
            if !matches!(key.as_str(), "enter" | "space") {
                continue;
            }
            let (raw, current) = read_owner(run, seq)?;
            if current.view.round != end.round
                || [current.view.cursor_x, current.view.cursor_y] != end.at.as_slice()
                || !current.has("0x118C6")
                || !current.has("0x16FAE")
            {
                continue;
            }
            let (previous_raw, previous) = read_owner(run, seq - 1)?;
            if previous.view.round != end.round || !previous.has("0x117F8") {
                continue;
            }
            if proven.is_some() {
                return Err(game_err!("END {} 有多個原版系統選單來源", end.seq));
            }
            proven = Some((
                seq,
                super::chapter_parity_replay_test::sha256_hex_of(&previous_raw),
                super::chapter_parity_replay_test::sha256_hex_of(&raw),
            ));
        }
        let (seq, cursor, system) = proven.ok_or_else(|| game_err!("END {} 缺少原版系統選單來源", end.seq))?;
        end.before_seq = Some(seq);
        sources.push(serde_json::json!({
            "end_seq": end.seq, "before_seq": seq,
            "cursor_checkpoint_sha256": cursor, "system_checkpoint_sha256": system,
        }));
    }
    Ok((!sources.is_empty()).then_some(sources))
}
