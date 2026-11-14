//! Go 來源：`remake/cmd/fd2/assets_glob_native.go`（43 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! Go `filepath.Glob` 沒有 Rust 標準函式庫對應；呼叫端（`loadSprites`、`loadPortraits`、
//! `loadFIGANI`）的樣式只在最後一段含 `*`，所以這裡只實作「目錄固定、檔名含 `*`」：
//! 列出目錄、依檔名排序（同 Go `filepath.Glob` 經 `os.ReadDir` 的排序）後比對。
//! 樣式含 `?`、`[`、`\` 或目錄段含 `*` 時回空（失敗即關閉）。

use super::*;

/// 只含 `*` 的萬用字元比對；`*` 不跨過 `/`（Go `filepath.Match`）。
fn wildcard_match(pattern: &[u8], name: &[u8]) -> bool {
    match pattern.split_first() {
        None => name.is_empty(),
        Some((b'*', rest)) => (0..=name.len()).any(|i| !name[..i].contains(&b'/') && wildcard_match(rest, &name[i..])),
        Some((&c, rest)) => name.first() == Some(&c) && wildcard_match(rest, &name[1..]),
    }
}

/// Go `filepath.Glob(pattern)` 的子集（見模組文件）。
fn glob(pattern: &str) -> Vec<String> {
    let path = std::path::Path::new(pattern);
    let (Some(dir), Some(file)) = (path.parent(), path.file_name().and_then(|f| f.to_str())) else {
        return Vec::new();
    };
    let dir_text = dir.to_string_lossy();
    if dir_text.contains(['*', '?', '[', '\\']) || file.contains(['?', '[', '\\']) {
        return Vec::new();
    }
    let Ok(entries) = std::fs::read_dir(if dir_text.is_empty() { std::path::Path::new(".") } else { dir }) else {
        return Vec::new();
    };
    let mut names: Vec<String> = entries.filter_map(|e| e.ok()?.file_name().into_string().ok()).collect();
    names.sort();
    names
        .into_iter()
        .filter(|name| wildcard_match(file.as_bytes(), name.as_bytes()))
        .map(|name| dir.join(name).to_string_lossy().into_owned())
        .collect()
}

/// assetGlob 同 assetPath 的五層查找,用於萬用字元批次載入(sprite/portrait/figani)。
/// 第一層有命中(非空)就整層採用,不同層的檔案不混拼。
///
/// Go：`assets_glob_native.go:12`，`func assetGlob(pattern string) []string`
pub(crate) fn asset_glob(pattern: &str) -> Vec<String> {
    let m = glob(&join_path(&[&user_data_dir(), pattern]));
    if !m.is_empty() {
        return m;
    }
    if let Ok(appdir) = std::env::var("APPDIR")
        && !appdir.is_empty()
    {
        let m = glob(&join_path(&[&appdir, pattern]));
        if !m.is_empty() {
            return m;
        }
    }
    let resources = mac_bundle_resource_dir();
    if !resources.is_empty() {
        let m = glob(&join_path(&[&resources, pattern]));
        if !m.is_empty() {
            return m;
        }
    }
    let d = exe_dir();
    if !d.is_empty() {
        let m = glob(&join_path(&[&d, pattern]));
        if !m.is_empty() {
            return m;
        }
    }
    if let Ok(cwd) = std::env::current_dir() {
        let mut dir = cwd;
        for _ in 0..5 {
            let m = glob(&dir.join(pattern).to_string_lossy());
            if !m.is_empty() {
                return m;
            }
            match dir.parent() {
                Some(parent) if parent != dir => dir = parent.to_path_buf(),
                _ => break,
            }
        }
    }
    Vec::new()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn wildcard_match_follows_filepath_match_for_star() {
        assert!(wildcard_match(b"fig_*_f*.png", b"fig_001_f02.png"));
        assert!(!wildcard_match(b"fig_*_f*.png", b"fig_001_f02.gif"));
        assert!(!wildcard_match(b"a*b", b"a/b"));
        assert!(wildcard_match(b"*", b""));
    }

    #[test]
    fn glob_lists_sorted_matches_in_final_segment() {
        let dir = crate::test_support::temp_dir("glob");
        for name in ["fig_2_f1.png", "fig_1_f1.png", "other.png", "fig_1_f0.png"] {
            std::fs::write(dir.join(name), b"x").unwrap();
        }
        let got = glob(&dir.join("fig_*_f*.png").to_string_lossy());
        let names: Vec<_> =
            got.iter().map(|p| std::path::Path::new(p).file_name().unwrap().to_str().unwrap()).collect();
        assert_eq!(names, ["fig_1_f0.png", "fig_1_f1.png", "fig_2_f1.png"]);
        assert!(glob(&dir.join("*/x").to_string_lossy()).is_empty());
    }
}
