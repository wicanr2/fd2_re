package battle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// nativeCombatTables 是 docs/data/exe_tables/native_combat_tables.json 的最小視圖。
type nativeCombatTables struct {
	Tables struct {
		ClassCritPct struct {
			Values []int `json:"values"`
		} `json:"class_crit_pct"`
		TerrainAPPct struct {
			Values []int `json:"values"`
		} `json:"terrain_ap_pct"`
		TerrainDPPct struct {
			Values []int `json:"values"`
		} `json:"terrain_dp_pct"`
	} `json:"tables"`
}

func loadNativeCombatTables(t *testing.T) nativeCombatTables {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..",
		"docs", "data", "exe_tables", "native_combat_tables.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tables nativeCombatTables
	if err := json.Unmarshal(raw, &tables); err != nil {
		t.Fatal(err)
	}
	return tables
}

// TestNativeTerrainAPDPPctMatchesTheExeTables 釘住 NativeTerrainAPDPPct 就是原版
// 的 0x51A12／0x51A2A 兩張表。它們各只有 6 格，第 7 格起就是 FDTXT.DAT 的字串，
// 所以界外一定要失敗即關閉，不能沿用最後一格。
func TestNativeTerrainAPDPPctMatchesTheExeTables(t *testing.T) {
	tables := loadNativeCombatTables(t)
	ap := tables.Tables.TerrainAPPct.Values
	dp := tables.Tables.TerrainDPPct.Values
	if len(ap) != 6 || len(dp) != 6 {
		t.Fatalf("表長 ap=%d dp=%d，原版各 6 格", len(ap), len(dp))
	}
	for code := 0; code < 6; code++ {
		gotAP, gotDP, ok := NativeTerrainAPDPPct(byte(code))
		if !ok {
			t.Fatalf("地形碼 %d 被判成不認得", code)
		}
		if gotAP != ap[code] || gotDP != dp[code] {
			t.Fatalf("地形碼 %d 是 (%d,%d)，原版是 (%d,%d)",
				code, gotAP, gotDP, ap[code], dp[code])
		}
	}
	if _, _, ok := NativeTerrainAPDPPct(6); ok {
		t.Fatal("地形碼 6 在原版表外，必須失敗即關閉")
	}
}

// TestClassCritPctMatchesTheExeTable 釘住每個職業的暴擊率就是 0x5239B。
// 這個值原版是 sub_29F72 的暴擊擲骰門檻（還要加上武器 record +10 的加成，
// 那一段重製端尚未接，見 worklist remake-attack-missing-counterattack）。
func TestClassCritPctMatchesTheExeTable(t *testing.T) {
	tables := loadNativeCombatTables(t)
	crit := tables.Tables.ClassCritPct.Values
	if len(crit) < 24 {
		t.Fatalf("暴擊表只有 %d 格", len(crit))
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "..",
		"docs", "data", "exe_tables", "resist_crit.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Class   int    `json:"cls"`
		Name    string `json:"name"`
		CritPct int    `json:"crit_pct"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("resist_crit.json 是空的")
	}
	for _, row := range rows {
		index := row.Class - 1
		if index < 0 || index >= len(crit) {
			t.Fatalf("職業 %d（%s）落在 0x5239B 之外", row.Class, row.Name)
		}
		if crit[index] != row.CritPct {
			t.Fatalf("職業 %d（%s）暴擊率 %d，0x5239B[%d] 是 %d",
				row.Class, row.Name, row.CritPct, index, crit[index])
		}
	}
}
