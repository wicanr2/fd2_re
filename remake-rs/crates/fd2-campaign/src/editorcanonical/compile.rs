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

/// canonicalNode 只宣告編譯需要的欄位。importer 把無法無損映射的值原樣留在
/// extensions.legacy，所以節點的本體其實在那裡；canonical 自己的 map_id／
/// scenario_id／story_id 是跨文件參照，編回 legacy 時用不到。
///
/// Go：`compile.go:20`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct CanonicalNode {
    /// Go 標籤：`json:"node_id"`
    pub node_id: String,
    /// Go 標籤：`json:"type"`
    pub r#type: String,
    /// Go 標籤：`json:"next"`
    pub next: GoRawMessage,
    /// Go 標籤：`json:"on_win"`
    pub on_win: GoRawMessage,
    /// Go 標籤：`json:"on_lose"`
    pub on_lose: GoRawMessage,
    /// Go 標籤：`json:"asset_ids"`
    pub asset_ids: GoRawMessage,
    /// 這三個是跨文件參照（`map/map0`），由有損映射產生，編回 legacy 時用不到——
    /// 原值在 extensions.legacy。仍要宣告出來：解碼是嚴格的，欄位漏宣告會直接
    /// 失敗，而那正是想要的——未知欄位被默默忽略就是資料遺失的來源。
    /// Go 標籤：`json:"map_id"`
    pub map_id: String,
    /// Go 標籤：`json:"scenario_id"`
    pub scenario_id: String,
    /// Go 標籤：`json:"story_id"`
    pub story_id: String,
    /// Go 標籤：`json:"extensions"`
    pub extensions: CanonicalNodeExtensions,
}

/// Go：`compile.go:39`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct CanonicalCampaign {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"document_id"`
    pub document_id: String,
    /// Go 標籤：`json:"kind"`
    pub kind: String,
    /// Go 標籤：`json:"nodes"`
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
pub fn compile_campaign_json(root: &str) -> Result<Vec<u8>> {
    todo!("editorcanonical/compile.go:CompileCampaignJSON")
}

/// resolveNodeReference 把 canonical 的 node_id 換回 legacy key。非字串或查不到的
/// 值原樣保留——importer 遇到解析不到的目標時就是這樣處理的，編回去要對稱。
///
/// Go：`compile.go:136`，`func resolveNodeReference(value json.RawMessage, keyByNodeID map[string]string) (json.RawMessage, error)`
pub(crate) fn resolve_node_reference(
    value: &GoRawMessage,
    key_by_node_id: &BTreeMap<String, String>,
) -> Result<GoRawMessage> {
    todo!("editorcanonical/compile.go:resolveNodeReference")
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct CanonicalNodeExtensions {
    /// Go 標籤：`json:"legacy"`
    pub legacy: BTreeMap<String, GoRawMessage>,
    /// Go 標籤：`json:"legacy_key"`
    pub legacy_key: String,
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct CanonicalCampaignSource {
    /// Go 標籤：`json:"path"`
    pub path: String,
    /// Go 標籤：`json:"importer_version"`
    pub importer_version: String,
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct CanonicalCampaignExtensions {
    /// Go 標籤：`json:"legacy"`
    pub legacy: BTreeMap<String, GoRawMessage>,
}
