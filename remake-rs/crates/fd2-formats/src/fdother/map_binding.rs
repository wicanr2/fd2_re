//! 移植自 Go `map_binding.go`。

use super::native_flow_error::{NativeFlowResult, fixed};
use std::path::{Component, Path};

/// 只接受匯出的戰術地圖位置。`assets` 是 map0 有文件記載的舊別名；其他被接受的路徑
/// 最後一段必須正好是 `mapN`。這是綁定，不是檔名猜測。
///
/// Go 版取 `filepath.Base(filepath.Clean(path))`。這是純字面運算、不碰檔案系統，結果直接決定
/// 接受與否，所以 Rust 版以 `std::path::Component` 照 Go 的 `Clean` 規則字面化簡（`.` 去掉、
/// `x/..` 抵消、根目錄的 `..` 丟棄）後取最後一段。數字部分對應 `strconv.Atoi`：允許前導 `+`／`-`
/// 與前導零，超出 64 位元即拒收；`str::parse::<i64>` 的規則相同。
pub fn map_index_from_asset_path(path: &str) -> NativeFlowResult<i64> {
    let base = clean_base(path);
    if base == "assets" {
        return Ok(0);
    }
    if !base.starts_with("map") || base.len() == 3 {
        return fixed("fdother: map asset path lacks explicit map index");
    }
    match base[3..].parse::<i64>() {
        Ok(index) if index >= 0 => Ok(index),
        _ => fixed("fdother: invalid map asset index"),
    }
}

/// `filepath.Base(filepath.Clean(path))` 的字面等價。
fn clean_base(path: &str) -> String {
    let mut rooted = false;
    let mut stack: Vec<String> = Vec::new();
    for component in Path::new(path).components() {
        match component {
            // Windows 磁碟代號：Go 的 `Base` 先丟掉 volume name，這裡同樣略過。
            Component::Prefix(_) => {}
            Component::RootDir => rooted = true,
            Component::CurDir => {}
            Component::ParentDir => match stack.last() {
                Some(last) if last != ".." => {
                    stack.pop();
                }
                _ if rooted => {}
                _ => stack.push("..".to_string()),
            },
            Component::Normal(name) => stack.push(name.to_string_lossy().into_owned()),
        }
    }
    match stack.pop() {
        Some(last) => last,
        None if rooted => std::path::MAIN_SEPARATOR.to_string(),
        None => ".".to_string(),
    }
}

#[cfg(test)]
mod tests;
