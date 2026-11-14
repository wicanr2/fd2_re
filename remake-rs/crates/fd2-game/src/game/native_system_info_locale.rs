//! Go 來源：`remake/cmd/fd2/native_system_info_locale.go`（75 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_system_info_locale.go:12`
pub(crate) static SYSTEM_INFO_CAPTION_KEYS: LazyLock<[String; 8]> = LazyLock::new(|| {
    [
        String::from("map"),
        String::from("turn"),
        String::from("enemy"),
        String::from("friend"),
        String::from("npc"),
        String::from("currency"),
        String::from("victory"),
        String::from("defeat"),
    ]
});

/// Go：`native_system_info_locale.go:14`，`func init()`
///
/// Go 的 `init()` 在套件初始化時把 68 個 `battle.info.*` 鍵補進 `officialLocaleContract`。Rust 沒有套件
/// 初始化階段：A 組在 `OFFICIAL_LOCALE_CONTRACT` 的 `LazyLock` 初始式直接併入
/// [`system_info_locale_contract`]，所以第一次讀契約時就已含這些鍵。本函式保留 Go 的寫法（覆寫同名鍵），
/// 重複呼叫結果不變。
pub(crate) fn init_native_system_info_locale() {
    let mut contract = OFFICIAL_LOCALE_CONTRACT.write().unwrap();
    for (key, entry) in system_info_locale_contract() {
        contract.insert(key, entry);
    }
}

/// Go：`native_system_info_locale.go:20`，`func systemInfoLocaleContract() map[string]localeEntryContract`
pub(crate) fn system_info_locale_contract() -> BTreeMap<String, LocaleEntryContract> {
    let mut out = BTreeMap::new();
    for key in SYSTEM_INFO_CAPTION_KEYS.iter() {
        out.insert(
            format!("battle.info.{key}"),
            LocaleEntryContract { variables: Vec::new(), source_id: format!("FDOTHER_005/system-info/{key}") },
        );
    }
    for index in 597..657 {
        out.insert(
            format!("battle.info.condition.{index:04}"),
            LocaleEntryContract { variables: Vec::new(), source_id: format!("FDTXT_000/string_{index:04}") },
        );
    }
    out
}
