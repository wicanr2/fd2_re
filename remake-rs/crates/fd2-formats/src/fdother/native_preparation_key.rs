//! 移植自 Go `native_preparation_key.go`。

/// 保存 `0x32004` 觀察到的回傳位元組契約。原版 helper 先把 `[0x53a8e]` 設為 0x10，經
/// `0x36d98` 讀兩位元組的 raw 輸入記錄，再套用這些位元組分支：延伸碼 0xe0／0x52 與
/// `[0x53a8d]==0x20` 得 0x1c；否則 `[0x53a8e]==0x53` 得 1。刻意不命名動作，也不改整備名單；
/// 那些語意屬於 `0x31a29` 的呼叫端。
pub fn normalize_native_preparation_key(raw53a8d: u8, raw53a8e: u8) -> u8 {
    let mut raw53a8e = raw53a8e;
    if raw53a8e == 0xe0 || raw53a8e == 0x52 || raw53a8d == 0x20 {
        raw53a8e = 0x1c;
    }
    if raw53a8e == 0x53 {
        raw53a8e = 1;
    }
    raw53a8e
}

#[cfg(test)]
mod tests;
