package editorcanonical

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

const (
	bundleRoot = "../../assets/editor-canonical"
	legacyPath = "../../assets/scenarios/campaign_full.json"
)

// 編譯結果要和 legacy 原檔在 JSON 值域上完全相同。這是整條編輯器路徑唯一有意義
// 的驗收：只驗「編得出東西」或「節點數一樣」都會放過欄位遺失，而遺失的欄位要玩到
// 那個節點才會發作。
func TestCompiledCampaignEqualsLegacyDocument(t *testing.T) {
	compiled, err := CompileCampaignJSON(bundleRoot)
	if err != nil {
		t.Fatalf("編譯 canonical 戰役: %v", err)
	}
	legacy, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("讀 legacy 戰役: %v", err)
	}
	var fromCanonical, fromLegacy any
	if err := json.Unmarshal(compiled, &fromCanonical); err != nil {
		t.Fatalf("解析編譯結果: %v", err)
	}
	if err := json.Unmarshal(legacy, &fromLegacy); err != nil {
		t.Fatalf("解析 legacy 戰役: %v", err)
	}
	if reflect.DeepEqual(fromCanonical, fromLegacy) {
		return
	}
	// 不相等時把差異指到具體節點與欄位；只說「不相等」的失敗訊息沒辦法動手修。
	left, _ := fromCanonical.(map[string]any)
	right, _ := fromLegacy.(map[string]any)
	for key := range right {
		if !reflect.DeepEqual(left[key], right[key]) {
			if key != "nodes" {
				t.Errorf("頂層欄位 %q 不同", key)
				continue
			}
			leftNodes, _ := left[key].(map[string]any)
			rightNodes, _ := right[key].(map[string]any)
			reported := 0
			for node := range rightNodes {
				if reflect.DeepEqual(leftNodes[node], rightNodes[node]) {
					continue
				}
				if reported++; reported > 5 {
					t.Errorf("還有更多節點不同，共 %d 個", len(rightNodes))
					break
				}
				t.Errorf("節點 %q 不同：\n編譯=%v\nlegacy=%v", node, leftNodes[node], rightNodes[node])
			}
		}
	}
	for key := range left {
		if _, ok := right[key]; !ok {
			t.Errorf("編譯結果多出頂層欄位 %q", key)
		}
	}
	t.FailNow()
}

// 編譯結果要能走完 campaign 自己那道轉場驗證，而不是只有 JSON 長得像；順帶守住
// canonical 與 legacy 不漂移——任何一邊被單獨改到，就會出現「編輯器看到的」與
// 「遊戲跑的」不是同一份，而那種差異要玩到那個節點才會發作。
func TestCompiledCampaignDecodesAndMatchesLegacyGraph(t *testing.T) {
	compiled, err := CompileCampaignJSON(bundleRoot)
	if err != nil {
		t.Fatalf("編譯 canonical 戰役: %v", err)
	}
	fromCanonical, err := campaign.Decode(compiled)
	if err != nil {
		t.Fatalf("編譯結果過不了戰役驗證: %v", err)
	}
	fromLegacy, err := campaign.Load(legacyPath)
	if err != nil {
		t.Fatalf("讀 legacy 戰役: %v", err)
	}
	if fromCanonical.Start != fromLegacy.Start {
		t.Fatalf("起點 %q，legacy 是 %q", fromCanonical.Start, fromLegacy.Start)
	}
	if len(fromCanonical.Nodes) != len(fromLegacy.Nodes) {
		t.Fatalf("節點數 %d，legacy 是 %d", len(fromCanonical.Nodes), len(fromLegacy.Nodes))
	}
	if !reflect.DeepEqual(fromCanonical, fromLegacy) {
		for id, want := range fromLegacy.Nodes {
			got, ok := fromCanonical.Nodes[id]
			if !ok {
				t.Fatalf("編譯結果缺節點 %q", id)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("節點 %q 的型別化內容不同", id)
			}
		}
		t.Fatal("戰役圖不同，但逐節點比對沒找到差異；" +
			"改了 legacy 就要重跑 tools/export_editor_canonical.py")
	}
}

// 壞掉的 bundle 要說得出壞在哪，不能編出一份少了東西卻看起來正常的戰役。
func TestCompileRefusesBrokenBundles(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(bundleRoot, "campaign", "campaign_full.json"))
	if err != nil {
		t.Fatalf("讀 canonical 戰役: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(source, &document); err != nil {
		t.Fatalf("解析 canonical 戰役: %v", err)
	}

	for _, testCase := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"schema 版本不對", func(d map[string]any) { d["schema_version"] = 2 }},
		{"kind 不對", func(d map[string]any) { d["kind"] = "scenario" }},
		{"沒有節點", func(d map[string]any) { d["nodes"] = []any{} }},
		{"節點少了 legacy_key", func(d map[string]any) {
			nodes := d["nodes"].([]any)
			node := nodes[0].(map[string]any)
			delete(node["extensions"].(map[string]any), "legacy_key")
		}},
		{"node_id 重複", func(d map[string]any) {
			nodes := d["nodes"].([]any)
			first := nodes[0].(map[string]any)
			second := nodes[1].(map[string]any)
			second["node_id"] = first["node_id"]
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var copied map[string]any
			if err := json.Unmarshal(source, &copied); err != nil {
				t.Fatalf("複製文件: %v", err)
			}
			testCase.mutate(copied)
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "campaign"), 0o755); err != nil {
				t.Fatalf("建立目錄: %v", err)
			}
			encoded, err := json.Marshal(copied)
			if err != nil {
				t.Fatalf("編碼文件: %v", err)
			}
			target := filepath.Join(root, "campaign", "campaign_full.json")
			if err := os.WriteFile(target, encoded, 0o644); err != nil {
				t.Fatalf("寫入文件: %v", err)
			}
			if _, err := CompileCampaignJSON(root); err == nil {
				t.Fatal("壞掉的 bundle 編出了結果")
			}
		})
	}

	if _, err := CompileCampaignJSON(""); err == nil {
		t.Fatal("空路徑編出了結果")
	}
	if _, err := CompileCampaignJSON(t.TempDir()); err == nil {
		t.Fatal("空目錄編出了結果")
	}
}

// 正對照：完整 bundle 在同一個流程下必須成功。沒有這一條，上面那些「壞掉會失敗」
// 也可能只是因為流程本身根本不會成功。
func TestCompileAcceptsTheShippedBundle(t *testing.T) {
	compiled, err := CompileCampaignJSON(bundleRoot)
	if err != nil {
		t.Fatalf("完整 bundle 編不出結果: %v", err)
	}
	if len(compiled) == 0 {
		t.Fatal("編譯結果是空的")
	}
}
