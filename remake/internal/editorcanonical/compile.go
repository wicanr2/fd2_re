// 把 canonical 文件編回正式執行期讀得動的戰役圖。
//
// bundle 一直只被驗證，沒有人消費：遊戲讀的是 `assets/scenarios/campaign_full.json`，
// 所以編輯器改了 canonical 文件，玩起來完全沒有差別。這一支補的就是那一段。
//
// 編譯方式是**還原成同一份 legacy JSON**，再交給 `campaign.Decode` 走原本那道
// 驗證。不另外寫一條解析路徑：分兩條會讓「編輯器存得起來但遊戲讀不動」這種差異
// 藏在兩份程式碼的縫裡，而那種差異只有玩到那個節點才會發作。
package editorcanonical

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

// canonicalNode 只宣告編譯需要的欄位。importer 把無法無損映射的值原樣留在
// extensions.legacy，所以節點的本體其實在那裡；canonical 自己的 map_id／
// scenario_id／story_id 是跨文件參照，編回 legacy 時用不到。
type canonicalNode struct {
	NodeID   string          `json:"node_id"`
	Type     string          `json:"type"`
	Next     json.RawMessage `json:"next"`
	OnWin    json.RawMessage `json:"on_win"`
	OnLose   json.RawMessage `json:"on_lose"`
	AssetIDs json.RawMessage `json:"asset_ids"`
	// 這三個是跨文件參照（`map/map0`），由有損映射產生，編回 legacy 時用不到——
	// 原值在 extensions.legacy。仍要宣告出來：解碼是嚴格的，欄位漏宣告會直接
	// 失敗，而那正是想要的——未知欄位被默默忽略就是資料遺失的來源。
	MapID      string `json:"map_id"`
	ScenarioID string `json:"scenario_id"`
	StoryID    string `json:"story_id"`
	Extensions struct {
		Legacy    map[string]json.RawMessage `json:"legacy"`
		LegacyKey string                     `json:"legacy_key"`
	} `json:"extensions"`
}

type canonicalCampaign struct {
	SchemaVersion int             `json:"schema_version"`
	DocumentID    string          `json:"document_id"`
	Kind          string          `json:"kind"`
	Nodes         []canonicalNode `json:"nodes"`
	Source        struct {
		Path             string `json:"path"`
		ImporterVersion  string `json:"importer_version"`
	} `json:"source"`
	Extensions struct {
		Legacy map[string]json.RawMessage `json:"legacy"`
	} `json:"extensions"`
}

// CompileCampaignJSON 讀 bundle 裡的 canonical 戰役文件，還原成 legacy 形狀的
// JSON。回傳位元組而不是型別，是為了讓驗收能直接比對兩份 JSON 的值——那比逐欄位
// 比對結構更難騙過去。
func CompileCampaignJSON(root string) ([]byte, error) {
	if root == "" {
		return nil, fmt.Errorf("editor canonical root is unavailable")
	}
	path := filepath.Join(root, "campaign", "campaign_full.json")
	var document canonicalCampaign
	if err := decodeStrict(path, &document); err != nil {
		return nil, fmt.Errorf("compile campaign: %w", err)
	}
	if document.SchemaVersion != 1 || document.Kind != "campaign" {
		return nil, fmt.Errorf("compile campaign: schema_version=%d kind=%q",
			document.SchemaVersion, document.Kind)
	}
	if len(document.Nodes) == 0 {
		return nil, fmt.Errorf("compile campaign: 文件沒有節點")
	}

	// 轉場目標在 canonical 裡是穩定 node_id；編回去要換成 legacy key。找不到對應
	// 的不是錯誤——importer 對解析不到的跨文件參照會保留原值並記一筆診斷。
	keyByNodeID := make(map[string]string, len(document.Nodes))
	for _, node := range document.Nodes {
		if node.Extensions.LegacyKey == "" {
			return nil, fmt.Errorf("compile campaign: 節點 %q 沒有 legacy_key", node.NodeID)
		}
		if node.NodeID == "" {
			return nil, fmt.Errorf("compile campaign: 節點 %q 沒有 node_id", node.Extensions.LegacyKey)
		}
		if _, seen := keyByNodeID[node.NodeID]; seen {
			return nil, fmt.Errorf("compile campaign: node_id %q 重複", node.NodeID)
		}
		keyByNodeID[node.NodeID] = node.Extensions.LegacyKey
	}

	nodes := make(map[string]map[string]json.RawMessage, len(document.Nodes))
	for _, node := range document.Nodes {
		if _, seen := nodes[node.Extensions.LegacyKey]; seen {
			return nil, fmt.Errorf("compile campaign: legacy_key %q 重複", node.Extensions.LegacyKey)
		}
		fields := make(map[string]json.RawMessage, len(node.Extensions.Legacy)+5)
		for field, value := range node.Extensions.Legacy {
			fields[field] = value
		}
		encodedType, err := json.Marshal(node.Type)
		if err != nil {
			return nil, fmt.Errorf("compile campaign: 節點 %q 的 type: %w", node.Extensions.LegacyKey, err)
		}
		fields["type"] = encodedType
		for field, value := range map[string]json.RawMessage{
			"next": node.Next, "on_win": node.OnWin, "on_lose": node.OnLose,
		} {
			if len(value) == 0 {
				continue
			}
			resolved, err := resolveNodeReference(value, keyByNodeID)
			if err != nil {
				return nil, fmt.Errorf("compile campaign: 節點 %q 的 %s: %w",
					node.Extensions.LegacyKey, field, err)
			}
			fields[field] = resolved
		}
		if len(node.AssetIDs) > 0 {
			fields["asset_ids"] = node.AssetIDs
		}
		nodes[node.Extensions.LegacyKey] = fields
	}

	restored := make(map[string]json.RawMessage, len(document.Extensions.Legacy)+1)
	for field, value := range document.Extensions.Legacy {
		restored[field] = value
	}
	encodedNodes, err := json.Marshal(nodes)
	if err != nil {
		return nil, fmt.Errorf("compile campaign: 編碼節點表: %w", err)
	}
	restored["nodes"] = encodedNodes
	return json.Marshal(restored)
}

// resolveNodeReference 把 canonical 的 node_id 換回 legacy key。非字串或查不到的
// 值原樣保留——importer 遇到解析不到的目標時就是這樣處理的，編回去要對稱。
func resolveNodeReference(value json.RawMessage, keyByNodeID map[string]string) (json.RawMessage, error) {
	var target string
	if err := json.Unmarshal(value, &target); err != nil {
		return value, nil
	}
	key, ok := keyByNodeID[target]
	if !ok {
		return value, nil
	}
	encoded, err := json.Marshal(key)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}
