//! 移植自 Go `native_preparation_cap.go`。

/// 重現 `0x318ad` 的 raw 上限閘門：原版章節／索引全域與 0x1a 比較，26（含）以下上限 15，
/// 超過時上限 19。參數是原版索引，不是顯示用的章數。
pub fn native_preparation_party_limit(native_index: i64) -> i64 {
    if native_index > 0x1a {
        return 19;
    }
    15
}

#[cfg(test)]
mod tests;
