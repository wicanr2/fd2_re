//! 移植自 Go `localization/locale.go`。

use super::patterns::{font_path_pattern, key_pattern, locale_pattern, pack_id_pattern};
use super::{Result, decode_strict, err, go_quote, strict_decode_error};
use crate::go_filepath;
use fd2_rules::battle::null_as_default;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

pub const SCHEMA_VERSION: i64 = 1;

/// Go：`locale.go:22`
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Font {
    #[serde(deserialize_with = "null_as_default")]
    pub profile: String,
    #[serde(deserialize_with = "null_as_default")]
    pub path: String,
}

/// Go：`locale.go:27`
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Entry {
    #[serde(deserialize_with = "null_as_default")]
    pub text: String,
    #[serde(deserialize_with = "null_as_default")]
    pub variables: Vec<String>,
    #[serde(deserialize_with = "null_as_default")]
    pub source_string_id: String,
}

/// Go：`locale.go:33`
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Pack {
    #[serde(deserialize_with = "null_as_default")]
    pub schema_version: i64,
    #[serde(deserialize_with = "null_as_default")]
    pub pack_id: String,
    #[serde(deserialize_with = "null_as_default")]
    pub locale: String,
    #[serde(deserialize_with = "null_as_default")]
    pub kind: String,
    /// Go 標籤：`base_locale,omitempty`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub base_locale: String,
    #[serde(deserialize_with = "null_as_default")]
    pub layout_profile: String,
    #[serde(deserialize_with = "null_as_default")]
    pub font: Font,
    #[serde(deserialize_with = "null_as_default")]
    pub entries: BTreeMap<String, Entry>,
}

/// Catalog is the immutable, validated view used by the runtime.
///
/// Go：`locale.go:45`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Catalog {
    pub locale: String,
    pub kind: String,
    pub base_locale: String,
    pub layout_profile: String,
    pub font: Font,
    pub entries: BTreeMap<String, Entry>,
}

/// Load reads one pack.json and rejects malformed or future fields.
///
/// Go：`locale.go:55`。讀檔錯誤的內層文字是 Rust `std::io` 的（見模組文件）。
pub fn load(path: &str) -> Result<Catalog> {
    let raw = std::fs::read(path).map_err(|e| err(format!("read locale pack {}: {e}", go_quote(path))))?;
    let pack: Pack =
        decode_strict(&raw).map_err(|e| strict_decode_error(&format!("decode locale pack {}", go_quote(path)), e))?;
    validate_pack(&pack).map_err(|e| e.wrap(format!("validate locale pack {}: ", go_quote(path))))?;
    // Go 逐筆複製 Variables（`append([]string(nil), ...)`）；Rust 的 clone 本來就是獨立的。
    Ok(Catalog {
        locale: pack.locale,
        kind: pack.kind,
        base_locale: pack.base_locale,
        layout_profile: pack.layout_profile,
        font: pack.font,
        entries: pack.entries,
    })
}

/// LoadOfficial loads <root>/<locale>/pack.json and requires an official pack.
///
/// Go：`locale.go:84`
pub fn load_official(root: &str, locale: &str) -> Result<Catalog> {
    if !locale_pattern(locale) {
        return Err(err(format!("invalid locale {}", go_quote(locale))));
    }
    let catalog = load(&go_filepath::join(&[root, locale, "pack.json"]))?;
    if catalog.kind != "official" || catalog.locale != locale {
        return Err(err(format!(
            "pack identity mismatch: kind={} locale={}, want official/{}",
            go_quote(&catalog.kind),
            go_quote(&catalog.locale),
            go_quote(locale)
        )));
    }
    Ok(catalog)
}

/// `Catalog::format` 的參數（Go `args ...any`）。語系文字的呼叫端只傳整數、字串與浮點數。
#[derive(Debug, Clone, PartialEq)]
pub enum FormatArg {
    Int(i64),
    Str(String),
    Float(f64),
}

impl From<i64> for FormatArg {
    fn from(v: i64) -> Self {
        FormatArg::Int(v)
    }
}

impl From<i32> for FormatArg {
    fn from(v: i32) -> Self {
        FormatArg::Int(v as i64)
    }
}

impl From<&str> for FormatArg {
    fn from(v: &str) -> Self {
        FormatArg::Str(v.to_string())
    }
}

impl From<String> for FormatArg {
    fn from(v: String) -> Self {
        FormatArg::Str(v)
    }
}

impl From<f64> for FormatArg {
    fn from(v: f64) -> Self {
        FormatArg::Float(v)
    }
}

impl Catalog {
    /// Format substitutes exactly the declared printf-style variables.
    ///
    /// Go：`locale.go:98`。Go 以 `fmt.Sprintf` 代入後檢查輸出有沒有 `%!`；Rust 用 [`go_sprintf`]。
    pub fn format(&self, key: &str, args: &[FormatArg]) -> Result<String> {
        let Some(entry) = self.entries.get(key) else {
            return Err(err(format!("unknown locale key {}", go_quote(key))));
        };
        if args.len() != entry.variables.len() {
            return Err(err(format!(
                "locale key {} expects {} arguments, got {}",
                go_quote(key),
                entry.variables.len(),
                args.len()
            )));
        }
        let formatted = go_sprintf(&entry.text, args);
        if formatted.contains("%!") {
            return Err(err(format!("locale key {} received incompatible argument types", go_quote(key))));
        }
        Ok(formatted)
    }
}

/// Go：`locale.go:117`
fn validate_pack(p: &Pack) -> Result<()> {
    if p.schema_version != SCHEMA_VERSION {
        return Err(err(format!("unsupported schema_version {}", p.schema_version)));
    }
    if !pack_id_pattern(&p.pack_id) {
        return Err(err("pack_id is invalid"));
    }
    if !locale_pattern(&p.locale) {
        return Err(err(format!("invalid locale {}", go_quote(&p.locale))));
    }
    if p.kind != "official" && p.kind != "community" {
        return Err(err(format!("invalid kind {}", go_quote(&p.kind))));
    }
    if p.kind == "official" && !p.base_locale.is_empty() {
        return Err(err("official pack must not declare base_locale"));
    }
    if p.kind == "community" {
        if !locale_pattern(&p.base_locale) {
            return Err(err("community pack requires valid base_locale"));
        }
        if p.base_locale == p.locale {
            return Err(err("community base_locale must differ from locale"));
        }
    }
    if p.layout_profile != "dos-8x8-latin" && p.layout_profile != "pc98-16x16-cjk" {
        return Err(err(format!("invalid layout_profile {}", go_quote(&p.layout_profile))));
    }
    if p.font.profile != "fd2-latin-8x8" && p.font.profile != "fd2-cjk-16x16" {
        return Err(err(format!("invalid font.profile {}", go_quote(&p.font.profile))));
    }
    validate_relative_path(&p.font.path).map_err(|e| e.wrap("font.path: "))?;
    if p.entries.is_empty() {
        return Err(err("entries must not be empty"));
    }
    // Go 以 map 的隨機順序走訪；多筆不合法時回報哪一筆由順序決定，Rust 依鍵排序（見模組文件）。
    for (key, entry) in &p.entries {
        if !key_pattern(key) || key.starts_with("legacy.") {
            return Err(err(format!("invalid semantic key {}", go_quote(key))));
        }
        if entry.text.is_empty() {
            return Err(err(format!("entry {} has empty text", go_quote(key))));
        }
        if entry.source_string_id.is_empty() {
            return Err(err(format!("entry {} has empty source_string_id", go_quote(key))));
        }
        let placeholders = format_placeholders(&entry.text);
        if placeholders != entry.variables {
            return Err(err(format!("entry {} variables do not match text placeholders", go_quote(key))));
        }
    }
    Ok(())
}

/// Go：`locale.go:163`。`filepath.IsAbs` 在 Go 的 Linux 版是「以 `/` 開頭」；`fonts/` 前綴已排除絕對路徑。
fn validate_relative_path(path: &str) -> Result<()> {
    if path.is_empty()
        || path.contains('\0')
        || path.starts_with('/')
        || path.contains('\\')
        || !font_path_pattern(path)
    {
        return Err(err("must be a non-empty relative slash path"));
    }
    let clean = go_filepath::clean(path);
    if clean == "." || clean == ".." || clean.starts_with("../") {
        return Err(err("path escapes pack root"));
    }
    Ok(())
}

/// Go：`locale.go:174`
pub(crate) fn format_placeholders(text: &str) -> Vec<String> {
    let b = text.as_bytes();
    let mut out = Vec::new();
    let mut i = 0;
    while i < b.len() {
        if b[i] != b'%' {
            i += 1;
            continue;
        }
        if i + 1 < b.len() && b[i + 1] == b'%' {
            i += 2;
            continue;
        }
        let start = i;
        i += 1;
        while i < b.len() && b"+-# 0".contains(&b[i]) {
            i += 1;
        }
        while i < b.len() && b[i].is_ascii_digit() {
            i += 1;
        }
        if i < b.len() && b[i] == b'.' {
            i += 1;
            while i < b.len() && b[i].is_ascii_digit() {
                i += 1;
            }
        }
        if i < b.len() && b"vTtbcdoOqxXUeEfFgGsxp".contains(&b[i]) {
            out.push(text[start..i + 1].to_string());
        }
        // Go 迴圈的 i++ 在這裡：停在動詞（或不認得的字元）上，下一輪從它後面開始。
        i += 1;
    }
    out
}

/// Go `fmt.Sprintf` 的子集，只服務 [`Catalog::format`] 的「代入後檢查有沒有 `%!`」契約。
///
/// 支援：`%%`；旗標 `+ - # 空格 0`、寬度、精度；`%d` 與 `%v`（整數）、`%x`／`%X`（整數與字串）、`%c`（整數）、
/// `%s`／`%v`／`%q`（字串）、`%f`／`%F`（浮點數）。動詞與參數型別不合、參數不夠或多出、`%` 後面沒有動詞
/// 時，輸出含 `%!`（Go 的錯誤標記；標記之後的文字不與 Go 逐字相同，呼叫端只看有沒有 `%!`）。
/// Go 支援但這裡不支援的寫法（其他動詞、`%[n]` 參數索引、`*` 寬度、浮點數的 `%v`／`%e`／`%g`）也輸出
/// `%!`，列在黃金差分的 `KNOWN_DIFFERENCES`；目前全部官方語系只用到 `%s`、`%d`、`%.0f`、`%02X`。
pub fn go_sprintf(text: &str, args: &[FormatArg]) -> String {
    let b = text.as_bytes();
    let mut out = String::with_capacity(text.len());
    let mut next = 0usize;
    let mut i = 0usize;
    let mut literal_start = 0usize;
    while i < b.len() {
        if b[i] != b'%' {
            i += 1;
            continue;
        }
        out.push_str(&text[literal_start..i]);
        i += 1;
        let mut spec = Spec::default();
        while i < b.len() {
            match b[i] {
                b'+' => spec.plus = true,
                // Go：`-` 同時清掉 `0`（不在右邊補零），`0` 只在沒有 `-` 時生效。
                b'-' => (spec.minus, spec.zero) = (true, false),
                b'#' => spec.sharp = true,
                b' ' => spec.space = true,
                b'0' => spec.zero = !spec.minus,
                _ => break,
            }
            i += 1;
        }
        let digits = |i: &mut usize| -> Option<usize> {
            let start = *i;
            while *i < b.len() && b[*i].is_ascii_digit() {
                *i += 1;
            }
            (*i > start).then(|| text[start..*i].parse().unwrap_or(usize::MAX))
        };
        spec.width = digits(&mut i);
        if i < b.len() && b[i] == b'.' {
            i += 1;
            spec.precision = Some(digits(&mut i).unwrap_or(0));
        }
        if i >= b.len() {
            out.push_str("%!(NOVERB)");
            literal_start = i;
            break;
        }
        let verb = text[i..].chars().next().expect("非空");
        i += verb.len_utf8();
        literal_start = i;
        if verb == '%' {
            out.push('%');
            continue;
        }
        let Some(arg) = args.get(next) else {
            out.push_str(&format!("%!{verb}(MISSING)"));
            continue;
        };
        next += 1;
        if verb == 'v' {
            // Go `doPrintf`：`%v` 把 `#` 換成 Go 語法旗標、`+` 換成結構欄位旗標，兩者原本的意思都清掉。
            (spec.sharp_v, spec.sharp) = (spec.sharp, false);
            spec.plus = false;
        }
        match format_one(verb, &spec, arg) {
            Some(s) => out.push_str(&s),
            None => out.push_str(&format!("%!{verb}({arg:?})")),
        }
    }
    out.push_str(&text[literal_start.min(text.len())..]);
    if next < args.len() {
        out.push_str("%!(EXTRA)");
    }
    out
}

#[derive(Default)]
struct Spec {
    plus: bool,
    minus: bool,
    sharp: bool,
    sharp_v: bool,
    space: bool,
    zero: bool,
    width: Option<usize>,
    precision: Option<usize>,
}

impl Spec {
    /// Go `fmt.pad`：寬度以 rune 計；`-` 靠左補空白，否則靠右補 `0` 旗標決定的填充字元（Go 1.22 的
    /// `writePadding` 對字串、字元與 `%q` 也補 `0`）。
    fn pad(&self, s: String) -> String {
        self.pad_with(s, if self.zero { "0" } else { " " })
    }

    fn pad_spaces(&self, s: String) -> String {
        self.pad_with(s, " ")
    }

    fn pad_with(&self, s: String, fill: &str) -> String {
        let n = s.chars().count();
        match self.width {
            Some(w) if w > n => {
                if self.minus {
                    s + &" ".repeat(w - n)
                } else {
                    fill.repeat(w - n) + &s
                }
            }
            _ => s,
        }
    }

    fn sign(&self, negative: bool) -> &'static str {
        if negative {
            "-"
        } else if self.plus {
            "+"
        } else if self.space {
            " "
        } else {
            ""
        }
    }

    /// Go `fmt.truncateString`：精度是最多幾個 rune。
    fn truncate(&self, s: &str) -> String {
        match self.precision {
            Some(p) => s.chars().take(p).collect(),
            None => s.to_string(),
        }
    }
}

/// Go `strconv.CanBackquote`：沒有反引號、沒有 Tab 以外的控制字元、沒有 U+FEFF。
fn can_backquote(s: &str) -> bool {
    s.chars().all(|c| c != '`' && c != '\u{feff}' && (c == '\t' || !(c < ' ' || c == '\x7f')))
}

/// Go `strconv.QuoteToASCII`（`%+q`）：在 [`go_quote`] 之外，把非 ASCII 字元寫成 `\u`／`\U` 跳脫。
fn go_quote_ascii(s: &str) -> String {
    let quoted = go_quote(s);
    let mut out = String::with_capacity(quoted.len());
    for c in quoted.chars() {
        match c as u32 {
            0..=0x7f => out.push(c),
            v @ 0x80..=0xffff => out.push_str(&format!("\\u{v:04x}")),
            v => out.push_str(&format!("\\U{v:08x}")),
        }
    }
    out
}

fn format_one(verb: char, spec: &Spec, arg: &FormatArg) -> Option<String> {
    match (verb, arg) {
        ('d' | 'v', FormatArg::Int(v)) => Some(format_int(spec, *v, 10, false)),
        ('x', FormatArg::Int(v)) => Some(format_int(spec, *v, 16, false)),
        ('X', FormatArg::Int(v)) => Some(format_int(spec, *v, 16, true)),
        ('x', FormatArg::Str(s)) => Some(format_string_hex(spec, s.as_bytes(), false)),
        ('X', FormatArg::Str(s)) => Some(format_string_hex(spec, s.as_bytes(), true)),
        ('c', FormatArg::Int(v)) => {
            // Go `fmtC`：轉成 uint64 後大於 utf8.MaxRune 的值（含全部負數）與代理區都是 U+FFFD。
            let c = u32::try_from(*v).ok().and_then(char::from_u32).unwrap_or('\u{fffd}');
            Some(spec.pad(c.to_string()))
        }
        ('v', FormatArg::Str(s)) if spec.sharp_v => Some(spec.pad(go_quote(&spec.truncate(s)))),
        ('s' | 'v', FormatArg::Str(s)) => Some(spec.pad(spec.truncate(s))),
        ('q', FormatArg::Str(s)) => {
            let s = spec.truncate(s);
            let quoted = if spec.sharp && can_backquote(&s) {
                format!("`{s}`")
            } else if spec.plus {
                go_quote_ascii(&s)
            } else {
                go_quote(&s)
            };
            Some(spec.pad(quoted))
        }
        ('f' | 'F', FormatArg::Float(v)) => Some(format_float(spec, *v)),
        _ => None,
    }
}

/// Go `fmt.fmtSbx`：字串逐位元組寫成兩位十六進位；精度限制位元組數；空格旗標在位元組之間加空白，
/// 配 `#` 時每個位元組都加 `0x`，只有 `#` 時整串加一次；空字串只補寬度。
fn format_string_hex(spec: &Spec, b: &[u8], upper: bool) -> String {
    let length = spec.precision.map_or(b.len(), |p| p.min(b.len()));
    let x = if upper { 'X' } else { 'x' };
    if length == 0 {
        return spec.pad(String::new());
    }
    let mut out = String::new();
    if spec.sharp {
        out.push('0');
        out.push(x);
    }
    for (i, byte) in b[..length].iter().enumerate() {
        if spec.space && i > 0 {
            out.push(' ');
            if spec.sharp {
                out.push('0');
                out.push(x);
            }
        }
        out.push_str(&if upper { format!("{byte:02X}") } else { format!("{byte:02x}") });
    }
    spec.pad(out)
}

/// Go `fmt.fmtFloat` 的 `%f`：預設精度 6；`#` 一定留小數點；`0` 旗標在符號之後補零到寬度；
/// NaN／Inf 不補零。小數的捨入兩邊都是對二進位值的正確捨入（同值取偶）。
fn format_float(spec: &Spec, v: f64) -> String {
    if v.is_nan() {
        let sign = if spec.plus {
            "+"
        } else if spec.space {
            " "
        } else {
            ""
        };
        return spec.pad_spaces(format!("{sign}NaN"));
    }
    if v.is_infinite() {
        return spec.pad_spaces(format!("{}Inf", spec.sign(v < 0.0)));
    }
    let mut body = format!("{:.*}", spec.precision.unwrap_or(6), v.abs());
    if spec.sharp && !body.contains('.') {
        body.push('.');
    }
    let sign = spec.sign(v.is_sign_negative());
    if spec.zero
        && let Some(w) = spec.width
    {
        let n = sign.len() + body.chars().count();
        if w > n {
            return format!("{sign}{}{body}", "0".repeat(w - n));
        }
    }
    spec.pad_spaces(format!("{sign}{body}"))
}

/// Go `fmt.fmtInteger` 的十進位與十六進位：精度是最少位數（精度 0 且值為 0 時不寫數字），沒有精度時
/// `0` 旗標補零到寬度；`#` 在十六進位前加 `0x`／`0X`。
fn format_int(spec: &Spec, v: i64, base: u32, upper: bool) -> String {
    let negative = v < 0;
    let magnitude = v.unsigned_abs();
    let mut digits = match (base, upper) {
        (16, false) => format!("{magnitude:x}"),
        (16, true) => format!("{magnitude:X}"),
        _ => magnitude.to_string(),
    };
    let sign = spec.sign(negative);
    // Go：有精度時精度就是最少位數；沒有精度但有 `0` 旗標與寬度時，位數補到「寬度減符號」，
    // `0x` 前綴另外加在前面（所以 `%#06x` 是 `0x0000ff`，前綴不算在寬度裡）。
    let min_digits = match spec.precision {
        // Go：精度 0 且值為 0 時什麼都不寫（沒有符號、沒有 0x），只補空白到寬度。
        Some(0) if magnitude == 0 => return spec.pad_spaces(String::new()),
        Some(p) => p,
        None if spec.zero && !spec.minus => spec.width.map_or(0, |w| w.saturating_sub(sign.len())),
        None => 0,
    };
    if digits.len() < min_digits {
        digits = "0".repeat(min_digits - digits.len()) + &digits;
    }
    let prefix = match (base, spec.sharp, upper) {
        (16, true, false) => "0x",
        (16, true, true) => "0X",
        _ => "",
    };
    // Go 在這裡先把 `0` 旗標關掉再 pad：整數的補零已經在位數裡，寬度剩下的都補空白。
    spec.pad_spaces(format!("{sign}{prefix}{digits}"))
}

#[cfg(test)]
mod locale_test;
