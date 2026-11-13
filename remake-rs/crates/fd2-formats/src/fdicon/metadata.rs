//! 分離素材包 `bank.json` 的欄位讀取（`serde_json::Value`）。
//!
//! 對格式正確的文件，結果與 Go `json.Unmarshal` 相同：未知欄位略過、缺少的欄位與 `null`
//! 視為零值（Go 對 int、string、struct 忽略 `null`，對 slice 設成 nil），陣列元素的 `null`
//! 也是零值。刻意不照做 Go 標準函式庫對格式錯誤或非常規文件的行為：欄位名稱只比對原字
//! （Go 另外不分大小寫比對）、重複鍵依 `serde_json` 處理、無效 UTF-8 直接拒收、型別不符的
//! 錯誤文字是這裡自己的。素材包由本專案工具產生並以雜湊驗證，這些情況不是原版遊戲行為。

use serde_json::{Map, Value};

#[derive(Debug, Clone, Default)]
pub(super) struct SourceDoc {
    pub file: String,
    pub size: i64,
    pub md5: String,
    pub sha256: String,
}

#[derive(Debug, Clone, Default)]
pub(super) struct SpriteDoc {
    pub index: i64,
    pub width: i64,
    pub height: i64,
    pub frame: String,
    pub mask: String,
    pub remap_mask: String,
}

/// 兩種文件共用的欄位，加上 FDSHAP 專用欄位（FDICON 文件中恆為零值）。
#[derive(Debug, Clone, Default)]
pub(super) struct Document {
    pub schema_version: i64,
    pub kind: String,
    pub asset_id: String,
    pub status: String,
    pub evidence: String,
    pub source: SourceDoc,
    pub sprites: Vec<SpriteDoc>,
    pub map_index: i64,
    pub image_resource: i64,
    pub control_resource: i64,
    pub controls: Vec<i64>,
}

type Fields<'a> = Option<&'a Map<String, Value>>;

fn wrong(key: &str) -> String {
    format!("json: field {key:?} has the wrong type")
}

/// `null` 與缺少都是零值；其餘必須是物件。
fn object<'a>(v: Option<&'a Value>, key: &str) -> Result<Fields<'a>, String> {
    match v {
        None | Some(Value::Null) => Ok(None),
        Some(Value::Object(m)) => Ok(Some(m)),
        Some(_) => Err(wrong(key)),
    }
}

fn int_value(v: Option<&Value>, key: &str) -> Result<i64, String> {
    match v {
        None | Some(Value::Null) => Ok(0),
        Some(v) => v.as_i64().ok_or_else(|| wrong(key)),
    }
}

fn int(m: Fields<'_>, key: &str) -> Result<i64, String> {
    int_value(m.and_then(|m| m.get(key)), key)
}

fn string(m: Fields<'_>, key: &str) -> Result<String, String> {
    match m.and_then(|m| m.get(key)) {
        None | Some(Value::Null) => Ok(String::new()),
        Some(Value::String(s)) => Ok(s.clone()),
        Some(_) => Err(wrong(key)),
    }
}

fn array<'a>(m: Fields<'a>, key: &str) -> Result<&'a [Value], String> {
    match m.and_then(|m| m.get(key)) {
        None | Some(Value::Null) => Ok(&[]),
        Some(Value::Array(a)) => Ok(a),
        Some(_) => Err(wrong(key)),
    }
}

pub(super) fn parse(raw: &[u8]) -> Result<Document, String> {
    let value: Value = serde_json::from_slice(raw).map_err(|e| format!("json: {e}"))?;
    let top = object(Some(&value), "document")?;
    let source = object(top.and_then(|m| m.get("source")), "source")?;
    let mut sprites = Vec::new();
    for entry in array(top, "sprites")? {
        let s = object(Some(entry), "sprites")?;
        sprites.push(SpriteDoc {
            index: int(s, "index")?,
            width: int(s, "width")?,
            height: int(s, "height")?,
            frame: string(s, "frame")?,
            mask: string(s, "mask")?,
            remap_mask: string(s, "remap_mask")?,
        });
    }
    let mut controls = Vec::new();
    for value in array(top, "controls")? {
        controls.push(int_value(Some(value), "controls")?);
    }
    Ok(Document {
        schema_version: int(top, "schema_version")?,
        kind: string(top, "kind")?,
        asset_id: string(top, "asset_id")?,
        status: string(top, "status")?,
        evidence: string(top, "evidence")?,
        source: SourceDoc {
            file: string(source, "file")?,
            size: int(source, "size")?,
            md5: string(source, "md5")?,
            sha256: string(source, "sha256")?,
        },
        sprites,
        map_index: int(top, "map_index")?,
        image_resource: int(top, "image_resource")?,
        control_resource: int(top, "control_resource")?,
        controls,
    })
}
