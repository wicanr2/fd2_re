//! Go 來源：`remake/cmd/fd2/player_controls_locale.go`（103 行，Go v.1.1.15 `7c710b34`）。R5 階段 4（r5i）移植。
//!
//! 按鍵設定面板的語系檔 `assets/locales/<語系>/controls.json`。面板是播放器外框，不是原版畫面，所以字串不放進
//! `pack.json`：`pack.json` 的鍵集合是 Go 與 Rust 兩個引擎共用的精確契約，多一個鍵另一邊就拒收整份語系。這裡另立一份
//! 同樣精確的契約：鍵集合與變數必須完全相同，缺鍵、多鍵或變數不符就整份拒收，不混用其他語系。
//!
//! JSON 用 serde_json 讀（`rust-port-no-go-stdlib`）；錯誤訊息的格式照 Go，內含的 JSON 解碼錯誤文字是 serde 的。

use super::*;

/// Go：`player_controls_locale.go:17`
pub(crate) const CONTROLS_LOCALE_KIND: &str = "fd2_controls_locale";

/// `controls.json` 的鍵與變數；動作名稱依 [`CONTROL_ACTIONS`] 產生。
///
/// Go：`player_controls_locale.go:20`，`func controlsLocaleContract() map[string][]string`
pub(crate) fn controls_locale_contract() -> BTreeMap<String, Vec<String>> {
    let mut contract = BTreeMap::new();
    for (key, variables) in [
        ("controls.title", &[][..]),
        ("controls.subtitle", &[]),
        ("controls.pad.disconnected", &[]),
        ("controls.pad.connected", &["%d"]),
        ("controls.column.action", &[]),
        ("controls.column.keyboard", &[]),
        ("controls.column.gamepad", &[]),
        ("controls.group.movement", &[]),
        ("controls.group.confirm", &[]),
        ("controls.group.system", &[]),
        ("controls.unbound", &[]),
        ("controls.dpad", &[]),
        ("controls.stick", &[]),
        ("controls.deadzone", &[]),
        ("controls.msg.ready", &[]),
        ("controls.msg.capture", &[]),
        ("controls.msg.conflict", &["%s", "%s"]),
        ("controls.msg.assigned", &[]),
        ("controls.msg.cleared", &[]),
        ("controls.msg.reset_row", &[]),
        ("controls.msg.reset_all", &[]),
        ("controls.msg.reserved", &[]),
        ("controls.msg.corrupt", &[]),
        ("controls.msg.partial", &[]),
        ("controls.msg.save_failed", &["%s"]),
        ("controls.help.keyboard", &[]),
        ("controls.help.gamepad", &[]),
        ("controls.reset_all", &[]),
    ] {
        contract.insert(key.to_string(), variables.iter().map(|v| v.to_string()).collect());
    }
    for action in &CONTROL_ACTIONS {
        contract.insert(format!("controls.action.{}", action.id), Vec::new());
    }
    contract
}

/// Go：`player_controls_locale.go:39`，`var controlsGroupKeys`
pub(crate) const CONTROLS_GROUP_KEYS: [&str; CONTROL_GROUP_COUNT] =
    ["controls.group.movement", "controls.group.confirm", "controls.group.system"];

/// Go：`player_controls_locale.go:41`，`type controlsLocaleEntry struct`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub(crate) struct ControlsLocaleEntry {
    #[serde(default)]
    pub(crate) text: String,
    /// Go `[]string`：`null` 或缺欄位是 nil，與空陣列在 `reflect.DeepEqual(append([]string{}, ...), want)` 下相同。
    #[serde(default, deserialize_with = "battle::null_as_default")]
    pub(crate) variables: Vec<String>,
}

/// Go：`player_controls_locale.go:46`，`type controlsLocaleFile struct`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub(crate) struct ControlsLocaleFile {
    #[serde(default)]
    pub(crate) schema_version: i64,
    #[serde(default)]
    pub(crate) kind: String,
    #[serde(default)]
    pub(crate) locale: String,
    #[serde(default, deserialize_with = "battle::null_as_default")]
    pub(crate) entries: BTreeMap<String, ControlsLocaleEntry>,
}

/// Go：`player_controls_locale.go:53`，`type controlsLocale struct`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ControlsLocale {
    pub(crate) locale: String,
    pub(crate) entries: BTreeMap<String, ControlsLocaleEntry>,
}

/// Go：`player_controls_locale.go:58`，`func loadControlsLocale(localeID string) (*controlsLocale, error)`
pub(crate) fn load_controls_locale(locale_id: &str) -> Result<ControlsLocale> {
    let path = join_path(&[&asset_path("assets/locales"), locale_id, "controls.json"]);
    load_controls_locale_from(&path, locale_id)
}

/// Go：`player_controls_locale.go:62`，`func loadControlsLocaleFrom(path, localeID string) (*controlsLocale, error)`
pub(crate) fn load_controls_locale_from(path: &str, locale_id: &str) -> Result<ControlsLocale> {
    let raw = std::fs::read(path).map_err(|e| game_err!("open {path}: {e}"))?;
    let file: ControlsLocaleFile =
        serde_json::from_slice(&raw).map_err(|e| game_err!("controls locale {locale_id}: {e}"))?;
    if file.schema_version != 1 || file.kind != CONTROLS_LOCALE_KIND || file.locale != locale_id {
        return Err(game_err!("controls locale {locale_id}: header mismatch"));
    }
    let contract = controls_locale_contract();
    if file.entries.len() != contract.len() {
        return Err(game_err!(
            "controls locale {locale_id} has {} keys, want exact {}",
            file.entries.len(),
            contract.len()
        ));
    }
    // Go 以 `range contract`（map，順序隨機）逐鍵核對；Rust 依鍵排序，只有同時缺多個鍵時回報的鍵可能不同。
    for (key, variables) in &contract {
        let Some(entry) = file.entries.get(key) else {
            return Err(game_err!("controls locale {locale_id} is missing {key}"));
        };
        if entry.text.is_empty() || entry.variables != *variables {
            return Err(game_err!("controls locale {locale_id} has invalid text or variables for {key}"));
        }
    }
    Ok(ControlsLocale { locale: locale_id.to_string(), entries: file.entries })
}

/// [`ControlsLocale::text`] 的參數（Go `args ...any`：面板只傳字串與整數）。
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum ControlsArg {
    S(String),
    D(i64),
}

impl ControlsLocale {
    /// 依契約格式化；語系檔不可用時回傳鍵名，讓缺件看得見，不改用其他語系。
    ///
    /// Go：`player_controls_locale.go:91`，`func (c *controlsLocale) text(key string, args ...any) string`。Go 以
    /// `fmt.Sprintf(entry.Text, args...)` 格式化；契約只允許 `%s`、`%d`，參數個數已核對，Rust 依序代入（`%%` 是 `%`）。
    pub(crate) fn text(this: Option<&ControlsLocale>, key: &str, args: &[ControlsArg]) -> String {
        let Some(c) = this else {
            return key.to_string();
        };
        let Some(entry) = c.entries.get(key) else {
            return key.to_string();
        };
        if entry.variables.len() != args.len() {
            return key.to_string();
        }
        if args.is_empty() {
            return entry.text.clone();
        }
        go_sprintf_sd(&entry.text, args)
    }
}

/// `fmt.Sprintf` 在 `%s`、`%d`、`%%` 三種動詞下的結果（契約只有這三種）。其他動詞照 Go 的格式寫成 `%!v(...)` 太繁瑣，
/// 契約核對已擋掉，這裡原樣輸出。
fn go_sprintf_sd(format: &str, args: &[ControlsArg]) -> String {
    let mut out = String::new();
    let mut args = args.iter();
    let mut chars = format.chars().peekable();
    while let Some(c) = chars.next() {
        if c != '%' {
            out.push(c);
            continue;
        }
        match chars.peek().copied() {
            Some('%') => {
                chars.next();
                out.push('%');
            }
            Some(verb @ ('s' | 'd')) => {
                chars.next();
                match args.next() {
                    Some(ControlsArg::S(s)) if verb == 's' => out.push_str(s),
                    Some(ControlsArg::D(d)) if verb == 'd' => out.push_str(&d.to_string()),
                    Some(ControlsArg::S(s)) => out.push_str(&format!("%!d(string={s})")),
                    Some(ControlsArg::D(d)) => out.push_str(&format!("%!s(int={d})")),
                    None => out.push_str(&format!("%!{verb}(MISSING)")),
                }
            }
            _ => out.push('%'),
        }
    }
    out
}
