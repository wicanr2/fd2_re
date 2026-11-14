//! Go `regexp` 樣式的逐字元比對版本（`locale.go`、`content.go` 的套件層變數）。
//!
//! 每個樣式都以 `^…$` 錨定整個字串（Go RE2 的 `$` 在非多行模式只比對文字結尾），而且各段字元類別
//! 互不重疊，比對結果唯一，不需要回溯。

fn all(s: &[u8], ok: impl Fn(u8) -> bool) -> bool {
    s.iter().all(|&b| ok(b))
}

fn lower(b: u8) -> bool {
    b.is_ascii_lowercase()
}

fn digit(b: u8) -> bool {
    b.is_ascii_digit()
}

/// `localePattern = ^[a-z]{2,3}(?:-[A-Z][a-z]{3})?(?:-[A-Z]{2}|-[0-9]{3})?$`
pub(crate) fn locale_pattern(s: &str) -> bool {
    let b = s.as_bytes();
    let lang = b.iter().take_while(|&&c| lower(c)).count();
    if !(2..=3).contains(&lang) {
        return false;
    }
    let mut rest = &b[lang..];
    // 文字（script）：-Xxxx
    if rest.len() >= 5 && rest[0] == b'-' && rest[1].is_ascii_uppercase() && all(&rest[2..5], lower) {
        rest = &rest[5..];
    }
    // 地區：-XX 或 -999
    match rest.len() {
        0 => true,
        3 => rest[0] == b'-' && all(&rest[1..], |c| c.is_ascii_uppercase()),
        4 => rest[0] == b'-' && all(&rest[1..], digit),
        _ => false,
    }
}

/// `keyPattern = ^[a-z][a-z0-9]*(?:\.[a-z0-9_]+)+$`
pub(crate) fn key_pattern(s: &str) -> bool {
    let mut parts = s.split('.');
    let Some(first) = parts.next() else {
        return false;
    };
    let fb = first.as_bytes();
    if fb.is_empty() || !lower(fb[0]) || !all(&fb[1..], |c| lower(c) || digit(c)) {
        return false;
    }
    let mut count = 0;
    for part in parts {
        let pb = part.as_bytes();
        if pb.is_empty() || !all(pb, |c| lower(c) || digit(c) || c == b'_') {
            return false;
        }
        count += 1;
    }
    count >= 1
}

/// `fontPathPattern = ^fonts/[a-z0-9][a-z0-9._/-]*\.(?:png|bin|json)$`
pub(crate) fn font_path_pattern(s: &str) -> bool {
    let Some(rest) = s.strip_prefix("fonts/") else {
        return false;
    };
    let r = rest.as_bytes();
    if r.is_empty() || !(lower(r[0]) || digit(r[0])) {
        return false;
    }
    if !all(r, |c| lower(c) || digit(c) || matches!(c, b'.' | b'_' | b'/' | b'-')) {
        return false;
    }
    // 第一個字元之後要能放下副檔名（中間的 `[a-z0-9._/-]*` 可以是空的）。
    [".png", ".bin", ".json"].iter().any(|ext| r.len() > ext.len() && rest.ends_with(ext))
}

/// `packIDPattern = ^[a-z0-9][a-z0-9._-]+$`
pub(crate) fn pack_id_pattern(s: &str) -> bool {
    let b = s.as_bytes();
    b.len() >= 2
        && (lower(b[0]) || digit(b[0]))
        && all(&b[1..], |c| lower(c) || digit(c) || matches!(c, b'.' | b'_' | b'-'))
}

/// `sha256Pattern = ^[0-9a-f]{64}$`
pub(crate) fn sha256_pattern(s: &str) -> bool {
    s.len() == 64 && all(s.as_bytes(), |c| digit(c) || (b'a'..=b'f').contains(&c))
}

/// `contentPlaceholderPattern.FindAllString(text, -1)`，樣式 `%(?:%|[-+0-9.#]*[a-zA-Z])`。
///
/// 由左往右找不重疊的比對：遇到 `%` 先試 `%%`，否則吃掉 `[-+0-9.#]*` 再要一個 ASCII 字母。類別裡沒有
/// 字母，所以貪婪吃完之後下一個字元不是字母時，較短的吃法也必然失敗；失敗就從下一個位元組繼續找。
pub(crate) fn content_placeholder_matches(text: &str) -> Vec<&str> {
    let b = text.as_bytes();
    let mut out = Vec::new();
    let mut i = 0;
    while i < b.len() {
        if b[i] != b'%' {
            i += 1;
            continue;
        }
        if i + 1 < b.len() && b[i + 1] == b'%' {
            out.push(&text[i..i + 2]);
            i += 2;
            continue;
        }
        let mut j = i + 1;
        while j < b.len() && matches!(b[j], b'-' | b'+' | b'0'..=b'9' | b'.' | b'#') {
            j += 1;
        }
        if j < b.len() && b[j].is_ascii_alphabetic() {
            out.push(&text[i..j + 1]);
            i = j + 1;
        } else {
            i += 1;
        }
    }
    out
}
