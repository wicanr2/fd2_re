//! Go `unicode` 套件的文字分類表與 `unicode.Is`／`unicode.In`（`rust-port-no-go-stdlib` 的具名例外）。
//!
//! 斷行規則（`font.go` 的 `runeBreakable`）用 `unicode.In(r, unicode.Han, unicode.Hiragana,
//! unicode.Katakana, unicode.Hangul)` 判斷字元能否自成一個斷行單位。這是 Unicode Script 屬性，Rust
//! 標準函式庫沒有對應；表格版本不同時，新收錄的字會改變換行位置，所以固定照 Go 1.22.12 的表
//! （Unicode 15.0.0）。只收斷行用到的四個 Script。
//!
//! 表格在 2026-11-14 以 `fd2-go-test-local` 容器列印 Go 的 `unicode.Han` 等 `RangeTable` 取得。
//! 每列是 `(lo, hi, stride)`，同 Go `unicode.Range16`／`Range32`；Go 把 16 位元與 32 位元範圍分開存，
//! 這裡合成一張依 `lo` 排序的表，成員判斷結果相同。

// 表格轉錄自 Go 1.22.12 src/unicode/tables.go（由 maketables 從 Unicode 15.0.0 字元資料庫產生）。
// 原始碼授權：
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// 字元資料來自 Unicode 字元資料庫，依 Unicode License v3 使用。

/// 一張 Go `unicode.RangeTable`：`(lo, hi, stride)`。
pub(crate) type RangeTable = &'static [(u32, u32, u32)];

/// Go `unicode.Han`。
pub(crate) static HAN: RangeTable = &[
    (0x2E80, 0x2E99, 1),
    (0x2E9B, 0x2EF3, 1),
    (0x2F00, 0x2FD5, 1),
    (0x3005, 0x3007, 2),
    (0x3021, 0x3029, 1),
    (0x3038, 0x303B, 1),
    (0x3400, 0x4DBF, 1),
    (0x4E00, 0x9FFF, 1),
    (0xF900, 0xFA6D, 1),
    (0xFA70, 0xFAD9, 1),
    (0x16FE2, 0x16FE3, 1),
    (0x16FF0, 0x16FF1, 1),
    (0x20000, 0x2A6DF, 1),
    (0x2A700, 0x2B739, 1),
    (0x2B740, 0x2B81D, 1),
    (0x2B820, 0x2CEA1, 1),
    (0x2CEB0, 0x2EBE0, 1),
    (0x2F800, 0x2FA1D, 1),
    (0x30000, 0x3134A, 1),
    (0x31350, 0x323AF, 1),
];

/// Go `unicode.Hiragana`。
pub(crate) static HIRAGANA: RangeTable = &[
    (0x3041, 0x3096, 1),
    (0x309D, 0x309F, 1),
    (0x1B001, 0x1B11F, 1),
    (0x1B132, 0x1B150, 30),
    (0x1B151, 0x1B152, 1),
    (0x1F200, 0x1F200, 1),
];

/// Go `unicode.Katakana`。
pub(crate) static KATAKANA: RangeTable = &[
    (0x30A1, 0x30FA, 1),
    (0x30FD, 0x30FF, 1),
    (0x31F0, 0x31FF, 1),
    (0x32D0, 0x32FE, 1),
    (0x3300, 0x3357, 1),
    (0xFF66, 0xFF6F, 1),
    (0xFF71, 0xFF9D, 1),
    (0x1AFF0, 0x1AFF3, 1),
    (0x1AFF5, 0x1AFFB, 1),
    (0x1AFFD, 0x1AFFE, 1),
    (0x1B000, 0x1B120, 288),
    (0x1B121, 0x1B122, 1),
    (0x1B155, 0x1B164, 15),
    (0x1B165, 0x1B167, 1),
];

/// Go `unicode.Hangul`。
pub(crate) static HANGUL: RangeTable = &[
    (0x1100, 0x11FF, 1),
    (0x302E, 0x302F, 1),
    (0x3131, 0x318E, 1),
    (0x3200, 0x321E, 1),
    (0x3260, 0x327E, 1),
    (0xA960, 0xA97C, 1),
    (0xAC00, 0xD7A3, 1),
    (0xD7B0, 0xD7C6, 1),
    (0xD7CB, 0xD7FB, 1),
    (0xFFA0, 0xFFBE, 1),
    (0xFFC2, 0xFFC7, 1),
    (0xFFCA, 0xFFCF, 1),
    (0xFFD2, 0xFFD7, 1),
    (0xFFDA, 0xFFDC, 1),
];

/// Go `unicode.Is(rangeTab, r)`：`r` 落在某一列的 `lo..=hi` 且與 `lo` 相差 `stride` 的倍數。負值與
/// 超出 `u32` 的值照 Go 回 `false`。
pub(crate) fn is(table: RangeTable, r: i32) -> bool {
    let Ok(r) = u32::try_from(r) else {
        return false;
    };
    table.iter().any(|&(lo, hi, stride)| r >= lo && r <= hi && (r - lo).is_multiple_of(stride))
}

/// Go `unicode.In(r, ranges...)`：`r` 屬於任一張表。
pub(crate) fn is_in(r: i32, tables: &[RangeTable]) -> bool {
    tables.iter().any(|&t| is(t, r))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn script_membership_follows_go_tables() {
        for r in ['漢', '〇', '々', '⺀'] {
            assert!(is(HAN, r as i32), "{r:?}");
        }
        // Go 表的 0x3005..0x3007 stride 2：U+3006 不屬於 Han。
        assert!(!is(HAN, 0x3006));
        assert!(is(HIRAGANA, 'あ' as i32) && is(KATAKANA, 'ア' as i32) && is(HANGUL, '한' as i32));
        // stride 30：0x1B132 與 0x1B150 屬於 Hiragana，中間的 0x1B140 不屬於。
        assert!(is(HIRAGANA, 0x1B132) && is(HIRAGANA, 0x1B150) && !is(HIRAGANA, 0x1B140));
        assert!(!is_in('A' as i32, &[HAN, HIRAGANA, KATAKANA, HANGUL]));
        assert!(!is_in(-1, &[HAN, HIRAGANA, KATAKANA, HANGUL]));
    }
}
