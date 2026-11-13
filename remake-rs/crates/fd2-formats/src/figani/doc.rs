//! 分離素材 JSON 的欄位讀取（`serde_json::Value`）。
//!
//! 刻意不照做：Go 版用 `encoding/json` 解進 struct，這裡不逐行重現它對畸形輸入的行為（大小寫不敏感的
//! 鍵、重複鍵原地合併、無效 UTF-8 換成 U+FFFD、錯誤文字等）。對格式正確的 JSON 採用相同的語意：
//! 缺欄位或 `null` 取零值、`null` 指標為 nil、型別不符即拒收；拒收的錯誤文字與 Go 不同，黃金資料只比
//! 接受／拒收。素材包由本專案工具產生並以雜湊核對，畸形輸入不是原版行為。

use serde_json::{Map, Value};

pub(crate) type Object = Map<String, Value>;

static EMPTY: std::sync::OnceLock<Object> = std::sync::OnceLock::new();

fn empty() -> &'static Object {
    EMPTY.get_or_init(Object::new)
}

/// 解析頂層物件；`null` 視為所有欄位皆為零值（Go 的 `json.Unmarshal` 對 `null` 不報錯）。
pub(crate) fn parse_object(raw: &[u8]) -> Result<Object, String> {
    match serde_json::from_slice::<Value>(raw).map_err(|e| e.to_string())? {
        Value::Object(map) => Ok(map),
        Value::Null => Ok(Object::new()),
        other => Err(format!("json: top-level value is {}", kind(&other))),
    }
}

fn kind(v: &Value) -> &'static str {
    match v {
        Value::Null => "null",
        Value::Bool(_) => "bool",
        Value::Number(_) => "number",
        Value::String(_) => "string",
        Value::Array(_) => "array",
        Value::Object(_) => "object",
    }
}

fn present<'a>(obj: &'a Object, key: &str) -> Option<&'a Value> {
    obj.get(key).filter(|v| !v.is_null())
}

fn mismatch(key: &str, want: &str, got: &Value) -> String {
    format!("json: field {key}: cannot use {} as {want}", kind(got))
}

pub(crate) fn int(obj: &Object, key: &str) -> Result<i64, String> {
    match present(obj, key) {
        None => Ok(0),
        Some(v) => v.as_i64().ok_or_else(|| mismatch(key, "int", v)),
    }
}

pub(crate) fn opt_int(obj: &Object, key: &str) -> Result<Option<i64>, String> {
    present(obj, key).map(|v| v.as_i64().ok_or_else(|| mismatch(key, "int", v))).transpose()
}

pub(crate) fn byte(obj: &Object, key: &str) -> Result<u8, String> {
    match present(obj, key) {
        None => Ok(0),
        Some(v) => v.as_u64().filter(|&n| n <= 0xff).map(|n| n as u8).ok_or_else(|| mismatch(key, "uint8", v)),
    }
}

pub(crate) fn string(obj: &Object, key: &str) -> Result<String, String> {
    match present(obj, key) {
        None => Ok(String::new()),
        Some(v) => v.as_str().map(str::to_string).ok_or_else(|| mismatch(key, "string", v)),
    }
}

pub(crate) fn boolean(obj: &Object, key: &str) -> Result<bool, String> {
    match present(obj, key) {
        None => Ok(false),
        Some(v) => v.as_bool().ok_or_else(|| mismatch(key, "bool", v)),
    }
}

pub(crate) fn opt_bool(obj: &Object, key: &str) -> Result<Option<bool>, String> {
    present(obj, key).map(|v| v.as_bool().ok_or_else(|| mismatch(key, "bool", v))).transpose()
}

/// 巢狀物件；缺欄位或 `null` 回傳空物件（所有欄位為零值）。
pub(crate) fn object<'a>(obj: &'a Object, key: &str) -> Result<&'a Object, String> {
    match present(obj, key) {
        None => Ok(empty()),
        Some(Value::Object(map)) => Ok(map),
        Some(v) => Err(mismatch(key, "object", v)),
    }
}

/// 物件陣列；元素為 `null` 時是零值元素。
pub(crate) fn objects<'a>(obj: &'a Object, key: &str) -> Result<Vec<&'a Object>, String> {
    array(obj, key)?
        .iter()
        .map(|v| match v {
            Value::Null => Ok(empty()),
            Value::Object(map) => Ok(map),
            other => Err(mismatch(key, "object", other)),
        })
        .collect()
}

pub(crate) fn array<'a>(obj: &'a Object, key: &str) -> Result<&'a [Value], String> {
    match present(obj, key) {
        None => Ok(&[]),
        Some(Value::Array(items)) => Ok(items),
        Some(v) => Err(mismatch(key, "array", v)),
    }
}

/// 陣列元素逐一轉換；`null` 元素取零值（與 Go 相同）。
pub(crate) fn ints(obj: &Object, key: &str) -> Result<Vec<i64>, String> {
    array(obj, key)?
        .iter()
        .map(|v| if v.is_null() { Ok(0) } else { v.as_i64().ok_or_else(|| mismatch(key, "int", v)) })
        .collect()
}

pub(crate) fn strings(obj: &Object, key: &str) -> Result<Vec<String>, String> {
    array(obj, key)?
        .iter()
        .map(|v| match v {
            Value::Null => Ok(String::new()),
            Value::String(s) => Ok(s.clone()),
            other => Err(mismatch(key, "string", other)),
        })
        .collect()
}

/// `map[string]int`；`null` 值取 0。
pub(crate) fn int_map(obj: &Object, key: &str) -> Result<std::collections::BTreeMap<String, i64>, String> {
    match present(obj, key) {
        None => Ok(Default::default()),
        Some(Value::Object(map)) => map
            .iter()
            .map(|(k, v)| {
                let n = if v.is_null() { Ok(0) } else { v.as_i64().ok_or_else(|| mismatch(key, "int", v)) };
                n.map(|n| (k.clone(), n))
            })
            .collect(),
        Some(v) => Err(mismatch(key, "object", v)),
    }
}
