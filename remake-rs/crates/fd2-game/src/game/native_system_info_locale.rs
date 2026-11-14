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
pub(crate) fn init_native_system_info_locale() {
    todo!("native_system_info_locale.go:init")
}

/// Go：`native_system_info_locale.go:20`，`func systemInfoLocaleContract() map[string]localeEntryContract`
pub(crate) fn system_info_locale_contract() -> BTreeMap<String, LocaleEntryContract> {
    todo!("native_system_info_locale.go:systemInfoLocaleContract")
}
