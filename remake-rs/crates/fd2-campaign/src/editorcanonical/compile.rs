//! Go 來源：`remake/internal/editorcanonical/compile.go`（150 行）。
//!
//! 把 canonical 文件編回正式執行期讀得動的戰役圖。
//!
//! bundle 一直只被驗證，沒有人消費：遊戲讀的是 `assets/scenarios/campaign_full.json`，
//! 所以編輯器改了 canonical 文件，玩起來完全沒有差別。這一支補的就是那一段。
//!
//! 編譯方式是**還原成同一份 legacy JSON**，再交給 `campaign.Decode` 走原本那道
//! 驗證。不另外寫一條解析路徑：分兩條會讓「編輯器存得起來但遊戲讀不動」這種差異
//! 藏在兩份程式碼的縫裡，而那種差異只有玩到那個節點才會發作。

use super::*;

use serde_json::value::RawValue;

/// Go `json.RawMessage` 的逐位元組對應：保存原始 JSON 片段的文字（解碼時原樣保留，含空白與鍵序）。
///
/// 模組根的 [`GoRawMessage`] 是骨架期的 `serde_json::Value` 別名（模組根不歸本組改）；本檔要逐位元組
/// 重現 Go `json.Marshal` 的輸出，所以直接用 `serde_json::value::RawValue`（`Cargo.toml` 已開
/// `raw_value`，PORTING.md「階段 2 開工前的決定」第 3 條）。
pub(crate) type RawMessage = Box<RawValue>;

/// serde 輔助：欄位出現時一律保存原始片段，連 `null` 也保存成 `null` 四個字元。
///
/// Go 的 `json.RawMessage` 實作 `Unmarshaler`，解碼 `null` 時呼叫 `UnmarshalJSON("null")`，所以
/// 「欄位缺席」（長度 0）與「明寫 `null`」不同；Rust 以 `None`／`Some("null")` 區分。
fn some_raw<'de, D: serde::Deserializer<'de>>(d: D) -> std::result::Result<Option<RawMessage>, D::Error> {
    <RawMessage as serde::Deserialize>::deserialize(d).map(Some)
}

/// canonicalNode 只宣告編譯需要的欄位。importer 把無法無損映射的值原樣留在
/// extensions.legacy，所以節點的本體其實在那裡；canonical 自己的 map_id／
/// scenario_id／story_id 是跨文件參照，編回 legacy 時用不到。
///
/// Go：`compile.go:20`
/// Rust：`json.RawMessage` 欄位改為 `Option<RawMessage>`（見 [`some_raw`]）；`RawValue` 沒有
/// `PartialEq`，所以只 derive `Debug`、`Clone`、`Default`。
#[derive(Debug, Clone, Default, serde::Deserialize)]
#[serde(default, deny_unknown_fields)]
pub(crate) struct CanonicalNode {
    /// Go 標籤：`json:"node_id"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub node_id: String,
    /// Go 標籤：`json:"type"`
    #[serde(rename = "type", deserialize_with = "fd2_rules::battle::null_as_default")]
    pub r#type: String,
    /// Go 標籤：`json:"next"`
    #[serde(deserialize_with = "some_raw")]
    pub next: Option<RawMessage>,
    /// Go 標籤：`json:"on_win"`
    #[serde(deserialize_with = "some_raw")]
    pub on_win: Option<RawMessage>,
    /// Go 標籤：`json:"on_lose"`
    #[serde(deserialize_with = "some_raw")]
    pub on_lose: Option<RawMessage>,
    /// Go 標籤：`json:"asset_ids"`
    #[serde(deserialize_with = "some_raw")]
    pub asset_ids: Option<RawMessage>,
    /// 這三個是跨文件參照（`map/map0`），由有損映射產生，編回 legacy 時用不到——
    /// 原值在 extensions.legacy。仍要宣告出來：解碼是嚴格的，欄位漏宣告會直接
    /// 失敗，而那正是想要的——未知欄位被默默忽略就是資料遺失的來源。
    /// Go 標籤：`json:"map_id"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub map_id: String,
    /// Go 標籤：`json:"scenario_id"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub scenario_id: String,
    /// Go 標籤：`json:"story_id"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub story_id: String,
    /// Go 標籤：`json:"extensions"`
    pub extensions: CanonicalNodeExtensions,
}

/// Go：`compile.go:39`
#[derive(Debug, Clone, Default, serde::Deserialize)]
#[serde(default, deny_unknown_fields)]
pub(crate) struct CanonicalCampaign {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"document_id"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub document_id: String,
    /// Go 標籤：`json:"kind"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub kind: String,
    /// Go 標籤：`json:"nodes"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub nodes: Vec<CanonicalNode>,
    /// Go 標籤：`json:"source"`
    pub source: CanonicalCampaignSource,
    /// Go 標籤：`json:"extensions"`
    pub extensions: CanonicalCampaignExtensions,
}

/// CompileCampaignJSON 讀 bundle 裡的 canonical 戰役文件，還原成 legacy 形狀的
/// JSON。回傳位元組而不是型別，是為了讓驗收能直接比對兩份 JSON 的值——那比逐欄位
/// 比對結構更難騙過去。
///
/// Go：`compile.go:56`，`func CompileCampaignJSON(root string) ([]byte, error)`
/// Rust：輸出與 Go `json.Marshal` 逐位元組相同：map 鍵排序、`RawMessage` 壓縮並做 HTML 跳脫
/// （見 [`go_compact`]、[`go_json_string`]）。
pub fn compile_campaign_json(root: &str) -> Result<Vec<u8>> {
    if root.is_empty() {
        return Err(campaign_err!("editor canonical root is unavailable"));
    }
    let path = std::path::Path::new(root).join("campaign").join("campaign_full.json");
    let mut document = CanonicalCampaign::default();
    decode_strict(&path.to_string_lossy(), &mut document).map_err(|err| err.wrap("compile campaign: "))?;
    if document.schema_version != 1 || document.kind != "campaign" {
        return Err(campaign_err!(
            "compile campaign: schema_version={} kind={}",
            document.schema_version,
            fd2_rules::battle::go_quote(&document.kind)
        ));
    }
    if document.nodes.is_empty() {
        return Err(campaign_err!("compile campaign: 文件沒有節點"));
    }

    // 轉場目標在 canonical 裡是穩定 node_id；編回去要換成 legacy key。找不到對應
    // 的不是錯誤——importer 對解析不到的跨文件參照會保留原值並記一筆診斷。
    let mut key_by_node_id: BTreeMap<String, String> = BTreeMap::new();
    for node in &document.nodes {
        if node.extensions.legacy_key.is_empty() {
            return Err(campaign_err!(
                "compile campaign: 節點 {} 沒有 legacy_key",
                fd2_rules::battle::go_quote(&node.node_id)
            ));
        }
        if node.node_id.is_empty() {
            return Err(campaign_err!(
                "compile campaign: 節點 {} 沒有 node_id",
                fd2_rules::battle::go_quote(&node.extensions.legacy_key)
            ));
        }
        if key_by_node_id.contains_key(&node.node_id) {
            return Err(campaign_err!("compile campaign: node_id {} 重複", fd2_rules::battle::go_quote(&node.node_id)));
        }
        key_by_node_id.insert(node.node_id.clone(), node.extensions.legacy_key.clone());
    }

    // 每個值是已壓縮的 JSON 文字（Go `json.Marshal` 對 `RawMessage` 做的事）。
    let mut nodes: BTreeMap<String, BTreeMap<String, String>> = BTreeMap::new();
    for node in &document.nodes {
        let key = &node.extensions.legacy_key;
        if nodes.contains_key(key) {
            return Err(campaign_err!("compile campaign: legacy_key {} 重複", fd2_rules::battle::go_quote(key)));
        }
        let mut fields: BTreeMap<String, String> = BTreeMap::new();
        for (field, value) in &node.extensions.legacy {
            fields.insert(field.clone(), go_compact(value.get()));
        }
        fields.insert("type".into(), go_json_string(&node.r#type));
        // Go 以 map 迭代 next／on_win／on_lose（`compile.go:103`）；寫入的是不同鍵，順序無關。
        for (field, value) in [("next", &node.next), ("on_win", &node.on_win), ("on_lose", &node.on_lose)] {
            let Some(value) = value else {
                continue;
            };
            let resolved = resolve_node_reference(value, &key_by_node_id).map_err(|err| {
                err.wrap(format!("compile campaign: 節點 {} 的 {field}: ", fd2_rules::battle::go_quote(key)))
            })?;
            fields.insert(field.into(), go_compact(resolved.get()));
        }
        if let Some(asset_ids) = &node.asset_ids {
            fields.insert("asset_ids".into(), go_compact(asset_ids.get()));
        }
        nodes.insert(key.clone(), fields);
    }

    let mut restored: BTreeMap<String, String> = BTreeMap::new();
    for (field, value) in &document.extensions.legacy {
        restored.insert(field.clone(), go_compact(value.get()));
    }
    let encoded_nodes = go_marshal_object(nodes.iter().map(|(k, fields)| (k, go_marshal_object(fields.iter()))));
    restored.insert("nodes".into(), encoded_nodes);
    Ok(go_marshal_object(restored.iter()).into_bytes())
}

/// resolveNodeReference 把 canonical 的 node_id 換回 legacy key。非字串或查不到的
/// 值原樣保留——importer 遇到解析不到的目標時就是這樣處理的，編回去要對稱。
///
/// Go：`compile.go:136`，`func resolveNodeReference(value json.RawMessage, keyByNodeID map[string]string) (json.RawMessage, error)`
/// Rust：Go 把 `null` 解碼進字串時不報錯、字串留空；空 node_id 在前面已被拒收，所以 `null` 原樣保留。
pub(crate) fn resolve_node_reference(
    value: &RawMessage,
    key_by_node_id: &BTreeMap<String, String>,
) -> Result<RawMessage> {
    let target = match serde_json::from_str::<Option<String>>(value.get()) {
        Ok(target) => target.unwrap_or_default(),
        Err(_) => return Ok(value.clone()),
    };
    let Some(key) = key_by_node_id.get(&target) else {
        return Ok(value.clone());
    };
    RawValue::from_string(go_json_string(key)).map_err(|err| CampaignError::new(err.to_string()))
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, serde::Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct CanonicalNodeExtensions {
    /// Go 標籤：`json:"legacy"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub legacy: BTreeMap<String, RawMessage>,
    /// Go 標籤：`json:"legacy_key"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub legacy_key: String,
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct CanonicalCampaignSource {
    /// Go 標籤：`json:"path"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub path: String,
    /// Go 標籤：`json:"importer_version"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub importer_version: String,
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, serde::Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct CanonicalCampaignExtensions {
    /// Go 標籤：`json:"legacy"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub legacy: BTreeMap<String, RawMessage>,
}

/// Go `json.Marshal(string)` 的輸出：`serde_json` 的字串跳脫（`"`、`\`、`\b`、`\f`、`\n`、`\r`、`\t`，
/// 其餘控制字元 `\u00xx`，與 Go 1.22 相同）再加上 Go 預設的 HTML 跳脫：`<`、`>`、`&` 寫成
/// `<`、`>`、`&`，U+2028／U+2029 寫成 ` `／` `。
pub(crate) fn go_json_string(s: &str) -> String {
    let quoted = serde_json::to_string(s).unwrap_or_default();
    let mut out = String::with_capacity(quoted.len());
    for c in quoted.chars() {
        match c {
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            c => out.push(c),
        }
    }
    out
}

/// Go `json.Marshal` 對 `RawMessage` 的處理（`compact` 且 `escapeHTML`）：去掉字串外的空白，
/// `<`、`>`、`&` 寫成 `<` 等，U+2028／U+2029 寫成 ` `／` `。輸入是已驗證的 JSON
/// （`RawValue` 解碼時檢查過），所以只需追蹤是否在字串內。
pub(crate) fn go_compact(raw: &str) -> String {
    let mut out = String::with_capacity(raw.len());
    let mut in_string = false;
    let mut escaped = false;
    for c in raw.chars() {
        if in_string {
            if escaped {
                escaped = false;
            } else if c == '\\' {
                escaped = true;
            } else if c == '"' {
                in_string = false;
            }
        } else if matches!(c, ' ' | '\t' | '\n' | '\r') {
            continue;
        } else if c == '"' {
            in_string = true;
        }
        match c {
            '<' => out.push_str("\\u003c"),
            '>' => out.push_str("\\u003e"),
            '&' => out.push_str("\\u0026"),
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            c => out.push(c),
        }
    }
    out
}

/// Go `json.Marshal(map[string]json.RawMessage)`：鍵依位元組排序（呼叫端以 `BTreeMap` 保證），
/// 值是已壓縮的 JSON 文字。
pub(crate) fn go_marshal_object<K: AsRef<str>, V: AsRef<str>>(entries: impl Iterator<Item = (K, V)>) -> String {
    let mut out = String::from("{");
    for (i, (key, value)) in entries.enumerate() {
        if i > 0 {
            out.push(',');
        }
        out.push_str(&go_json_string(key.as_ref()));
        out.push(':');
        out.push_str(value.as_ref());
    }
    out.push('}');
    out
}

// 測試：Go `remake/internal/editorcanonical/compile_test.go` 逐條移植。
#[cfg(test)]
mod compile_test;
