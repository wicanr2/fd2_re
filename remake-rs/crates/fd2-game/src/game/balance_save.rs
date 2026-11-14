//! Go 來源：`remake/cmd/fd2/balance_save.go`（174 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! F 組另外移植 `restoreBalanceSavedState`（Go 測試 `TestBalanceNativeCurrentRestoresEnemyBaseline…`
//! 直接呼叫它）。JSON 對應規則見 `save.rs` 的模組文件。

use super::*;

/// Go：`balance_save.go:13`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct BalanceSavedState {
    /// Go 標籤：`json:"options,omitempty"`
    #[serde(serialize_with = "go_json_balance_options::serialize", skip_serializing_if = "Option::is_none")]
    pub(crate) options: Option<battle::BalanceOptions>,
    /// Go 標籤：`json:"enemy_bases,omitempty"`
    #[serde(with = "go_json_int_key_map", skip_serializing_if = "BTreeMap::is_empty")]
    pub(crate) enemy_bases: BTreeMap<i64, battle::BalanceStats>,
}

/// Go：`balance_save.go:17`
///
/// Rust：`readBalanceSidecar` 先放 `Version: 1`、空的 `Entries` 再以 `json.Decoder` 解碼，JSON 沒寫的
/// 欄位保留這兩個值；serde 以容器層的 `default` 取得同樣的起點。
#[derive(Debug, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct BalanceSidecar {
    /// Go 標籤：`json:"version"`
    #[serde(deserialize_with = "sidecar_version")]
    pub(crate) version: i64,
    /// Go 標籤：`json:"entries"`
    ///
    /// Go 讀到 `null` 時 map 變 nil，`readBalanceSidecar` 最後再換回空 map；Rust 直接讀成空的。
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) entries: BTreeMap<String, BalanceSavedState>,
}

/// Go 讀 `"version": null` 時保留解碼前的 1。
fn sidecar_version<'de, D: serde::Deserializer<'de>>(d: D) -> std::result::Result<i64, D::Error> {
    use serde::Deserialize as _;
    Ok(Option::<i64>::deserialize(d)?.unwrap_or(1))
}

impl Default for BalanceSidecar {
    fn default() -> Self {
        Self { version: 1, entries: BTreeMap::new() }
    }
}

/// Go：`balance_save.go:22`，`func balanceSaveKey(plain []byte, slot int) (string, error)`
pub(crate) fn balance_save_key(plain: &[u8], slot: i64) -> Result<String> {
    use sha2::{Digest, Sha256};
    let hex = |b: &[u8]| -> String { Sha256::digest(b).iter().map(|x| format!("{x:02x}")).collect() };
    if plain.len() != fdsave::FILE_SIZE {
        return Err(game_err!("能力設定存檔大小不符"));
    }
    if slot == -1 {
        let end = fdsave::CURRENT_RUNTIME_HEADER_OFFSET + fdsave::CURRENT_RUNTIME_HEADER_SIZE;
        return Ok(format!("current:{}", hex(&plain[..end])));
    }
    let (a, b) = fdsave::slot_bounds(fdsave_slot(slot)?)?;
    Ok(format!("slot{slot}:{}", hex(&plain[a..b])))
}

/// Go：`balance_save.go:36`，`func readBalanceSidecar(path string) (balanceSidecar, error)`
///
/// Go 在錯誤時也回傳部分解碼的 `d`；所有呼叫端遇到錯誤都不用它，Rust 只回錯誤。
pub(crate) fn read_balance_sidecar(path: &str) -> Result<BalanceSidecar> {
    let d = BalanceSidecar::default();
    let file = match std::fs::File::open(format!("{path}.fd2-options.json")) {
        Ok(file) => file,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(d),
        Err(err) => return Err(GameError::new(err.to_string())),
    };
    let info = file.metadata().map_err(|err| GameError::new(err.to_string()))?;
    if info.len() > 1024 * 1024 {
        return Err(game_err!("能力設定附檔過大"));
    }
    // Go `json.NewDecoder(f).Decode(&d)`：只讀第一個 JSON 值，後面的內容不檢查。
    let mut stream = serde_json::Deserializer::from_reader(std::io::BufReader::new(file)).into_iter::<BalanceSidecar>();
    let d = match stream.next() {
        Some(Ok(d)) => d,
        Some(Err(err)) => return Err(GameError::new(err.to_string())),
        // Go 讀到空檔回 `io.EOF`。
        None => return Err(game_err!("EOF")),
    };
    if d.version != 1 || d.entries.len() > 10 {
        return Err(game_err!("能力設定附檔版本或筆數無效"));
    }
    // Go 以 map 迭代順序檢查；多筆錯誤同時存在時 Go 報哪一筆隨順序而定，Rust 依鍵的升冪。
    for v in d.entries.values() {
        if !v.enemy_bases.is_empty() && v.options.is_none() {
            return Err(game_err!("敵軍基準缺少倍率"));
        }
        if let Some(options) = &v.options {
            options.validate()?;
        }
        for (i, b) in &v.enemy_bases {
            if *i < 0
                || *i >= (fdsave::ROSTER_UNITS * 3) as i64
                || b.hp < 1
                || b.hp > 32767
                || b.mp < 0
                || b.mp > 32767
                || b.ap < 0
                || b.ap > 32767
                || b.dp < 0
                || b.dp > 32767
                || b.dx < 0
                || b.dx > 32767
            {
                return Err(game_err!("敵軍基準無效"));
            }
        }
    }
    Ok(d)
}

/// Go：`balance_save.go:79`，`func loadBalanceSavedState(path string, plain []byte, slot int) (balanceSavedState, error)`
pub(crate) fn load_balance_saved_state(path: &str, plain: &[u8], slot: i64) -> Result<BalanceSavedState> {
    let d = read_balance_sidecar(path)?;
    let key = balance_save_key(plain, slot)?;
    Ok(d.entries.get(&key).cloned().unwrap_or_default())
}

impl Game {
    /// Preserve both old and new fingerprints before replacing the native SAV.
    /// A crash at either rename still pairs a save only with its own options.
    ///
    /// Go：`balance_save.go:93`，`func (g *Game) replaceNativeSaveWithBalance(path string, stored []byte, slot int) error`
    pub(crate) fn replace_native_save_with_balance(&mut self, path: &str, stored: &[u8], slot: i64) -> Result<()> {
        let mut d = read_balance_sidecar(path)?;
        let old = std::fs::read(path).map_err(|err| GameError::new(err.to_string()))?;
        let old_plain = fdsave::decode(&old)?;
        let plain = fdsave::decode(stored)?;
        let mut keep = BTreeSet::new();
        for p in [&old_plain, &plain] {
            for s in -1..4 {
                keep.insert(balance_save_key(p, s)?);
            }
        }
        d.entries.retain(|k, _| keep.contains(k));
        let key = balance_save_key(&plain, slot)?;
        let mut state = BalanceSavedState { options: self.balance.clone(), ..Default::default() };
        if slot == -1
            && let Some(st) = &self.st
            && self.balance.as_ref().is_some_and(|b| b.enemy_percent != 100)
        {
            // Go 建立非 nil 的空 map；`omitempty` 寫出時 nil 與空相同。
            for (i, u) in st.units.iter().enumerate() {
                if u.camp == battle::ENEMY
                    && let Some(base) = u.enemy_balance_base
                {
                    state.enemy_bases.insert(i as i64, base);
                }
            }
        }
        // Go `g.balance.IsOriginal()`：nil 視為原版。
        if self.balance.as_ref().is_none_or(|b| b.is_original()) {
            d.entries.remove(&key);
        } else {
            d.entries.insert(key, state);
        }
        let sidecar = format!("{path}.fd2-options.json");
        if !d.entries.is_empty() || file_exists_balance(&sidecar) {
            let raw = go_json_marshal_indent(&d)?;
            write_save_file(&sidecar, &raw)?;
        }
        replace_native_current_save_atomic(path, stored)
    }
}

/// Go：`balance_save.go:155`，`func fileExistsBalance(path string) bool`
pub(crate) fn file_exists_balance(path: &str) -> bool {
    std::fs::metadata(path).is_ok()
}

impl Game {
    /// Go：`balance_save.go:157`，`func (g *Game) restoreBalanceSavedState(state balanceSavedState)`
    /// （F 組補移植，原標 `r8`）
    pub(crate) fn restore_balance_saved_state(&mut self, state: BalanceSavedState) {
        self.balance = state.options;
        self.balance_ui = BalanceEditor::default();
        let Some(st) = self.st.as_mut() else {
            return;
        };
        st.balance = self.balance.clone();
        // Go `g.balance.EnemyPercent`：有敵軍基準時讀檔端已驗證倍率存在；balance 為 nil 時 Go 會 panic。
        for (i, b) in &state.enemy_bases {
            if *i >= 0 && (*i as usize) < st.units.len() {
                let u = &mut st.units[*i as usize];
                if u.camp == battle::ENEMY {
                    u.enemy_balance_base = Some(*b);
                    u.enemy_balance_percent =
                        self.balance.as_ref().expect("balance_save.go:171 g.balance 為 nil").enemy_percent;
                }
            }
        }
    }
}

#[cfg(test)]
mod balance_save_test;
