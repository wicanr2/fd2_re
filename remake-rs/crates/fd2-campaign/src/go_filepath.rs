//! Go `path/filepath.Clean`／`Join` 的字面路徑處理（`rust-port-no-go-stdlib` 的具名例外）。
//!
//! Rust 標準函式庫沒有只看字面的路徑清理：`Path::components()` 會去掉 `.` 與重複的 `/`，但刻意保留
//! `..`（有符號連結時 `a/../b` 不一定等於 `b`），而且在 Windows 會把 `\` 與磁碟代號當成路徑語法，
//! 結果隨平台不同。語系包的路徑檢查要拒收 `fonts/../../x` 這種跳出包根目錄的寫法，需要跨平台一致、
//! 只認 `/` 的字面清理，所以照 Go 在 Unix 上的行為重寫。
//!
//! 呼叫端先拒收含 `\`、NUL 或以 `/` 開頭的路徑，所以 Go 在 Windows 上得到相同的接受或拒收結果。
//! 只有錯誤訊息裡的路徑字樣用到 [`join`]；不影響開檔的路徑拼接照一般規則用 `std::path`
//! （例如 `campaign::handler_binding` 的 `go_join_dir`）。

// 行為對照 Go 1.22.12 src/path/filepath/path.go 的 Clean 與 Join（Unix）。原始碼授權：
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/// Go `filepath.Clean`（Unix）：去掉空段與 `.`；`..` 吃掉前一個非 `..` 段；根目錄之上的 `..` 丟掉；
/// 結果是空字串時回 `.`。
pub(crate) fn clean(path: &str) -> String {
    let rooted = path.starts_with('/');
    let mut parts: Vec<&str> = Vec::new();
    for seg in path.split('/') {
        match seg {
            "" | "." => {}
            ".." => match parts.last() {
                Some(&last) if last != ".." => {
                    parts.pop();
                }
                _ if rooted => {}
                _ => parts.push(".."),
            },
            seg => parts.push(seg),
        }
    }
    let joined = parts.join("/");
    match (rooted, joined.is_empty()) {
        (true, _) => format!("/{joined}"),
        (false, true) => ".".to_string(),
        (false, false) => joined,
    }
}

/// Go `filepath.Join`：略過空字串，以 `/` 接起來再 [`clean`]；全部是空字串時回空字串。
pub(crate) fn join(parts: &[&str]) -> String {
    let parts: Vec<&str> = parts.iter().copied().filter(|p| !p.is_empty()).collect();
    if parts.is_empty() {
        return String::new();
    }
    clean(&parts.join("/"))
}

#[cfg(test)]
mod tests {
    use super::*;

    /// 期望值取自 Go `path/filepath` 的 `cleantests`（Unix 部分）。
    #[test]
    fn clean_matches_go_cleantests() {
        for (input, want) in [
            ("abc", "abc"),
            ("abc/def", "abc/def"),
            ("a/b/c", "a/b/c"),
            (".", "."),
            ("..", ".."),
            ("../..", "../.."),
            ("../../abc", "../../abc"),
            ("/abc", "/abc"),
            ("/", "/"),
            ("", "."),
            ("abc/", "abc"),
            ("/abc/", "/abc"),
            ("abc//def//ghi", "abc/def/ghi"),
            ("//abc", "/abc"),
            ("abc/./def", "abc/def"),
            ("/./abc/def", "/abc/def"),
            ("abc/.", "abc"),
            ("abc/def/ghi/../jkl", "abc/def/jkl"),
            ("abc/def/../ghi/../jkl", "abc/jkl"),
            ("abc/def/..", "abc"),
            ("abc/def/../..", "."),
            ("/abc/def/../..", "/"),
            ("abc/def/../../..", ".."),
            ("/abc/def/../../..", "/"),
            ("abc/def/../../../ghi/jkl/../../../mno", "../../mno"),
            ("/../abc", "/abc"),
            ("abc/./../def", "def"),
            ("abc//./../def", "def"),
            ("abc/../../././../def", "../../def"),
        ] {
            assert_eq!(clean(input), want, "clean({input:?})");
        }
    }

    #[test]
    fn join_skips_empty_parts() {
        assert_eq!(join(&[]), "");
        assert_eq!(join(&["", ""]), "");
        assert_eq!(join(&["a", "", "b"]), "a/b");
        assert_eq!(join(&["a/", "../b"]), "b");
        assert_eq!(join(&["", "/a"]), "/a");
    }
}
