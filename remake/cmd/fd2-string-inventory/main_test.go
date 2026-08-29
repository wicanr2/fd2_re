package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type reviewDocument struct {
	InventorySHA256 string `json:"inventory_sha256"`
	ReviewedRole    string `json:"reviewed_role"`
	Dispositions    map[string]struct {
		StringIDs []string `json:"string_ids"`
	} `json:"dispositions"`
}

func writeFixture(t *testing.T, root, relative, text string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildCollectsControlledJSONAndGoUIWithoutNotesOrErrors(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "remake/assets/story/ch00.json", `{
  "title":"序章", "notes":"研究註記", "lines":[
    {"speaker_name":"索爾", "text":"金幣 %d"}
  ]
}`)
	writeFixture(t, root, "remake/assets/scenarios/ch01.json", `{"name":"衛兵","label":"出擊"}`)
	writeFixture(t, root, "remake/assets/data/items.json", `[{"name":"藥草","raw":"原始證據"}]`)
	writeFixture(t, root, "remake/cmd/fd2/ui.go", `package main
func f() { g.msg = "按下 Enter"; drawText("ASCII UI"); fmt.Errorf("內部錯誤") }
`)

	inventory, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	texts := map[string]Entry{}
	for _, entry := range inventory.Entries {
		texts[entry.Text] = entry
	}
	for _, wanted := range []string{"序章", "索爾", "金幣 %d", "衛兵", "出擊", "藥草", "按下 Enter", "ASCII UI"} {
		if _, ok := texts[wanted]; !ok {
			t.Errorf("missing %q", wanted)
		}
	}
	for _, excluded := range []string{"研究註記", "原始證據", "內部錯誤"} {
		if _, ok := texts[excluded]; ok {
			t.Errorf("unexpected %q", excluded)
		}
	}
	if got := texts["金幣 %d"].Variables; !reflect.DeepEqual(got, []string{"%d"}) {
		t.Fatalf("variables=%v", got)
	}
}

func TestBuildIsDeterministicAndJSONSerializable(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "remake/assets/story/ch00.json", `{"text":"測試"}`)
	writeFixture(t, root, "remake/cmd/fd2/ui.go", `package main
func f() { draw("GAME OVER") }
`)
	first, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(first)
	right, _ := json.Marshal(second)
	if !reflect.DeepEqual(left, right) {
		t.Fatal("inventory output is not deterministic")
	}
}

func TestBuildRejectsDuplicateExistingStringID(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "remake/assets/story/ch00.json", `{
  "lines":[
    {"string_id":"story.same", "text":"甲"},
    {"string_id":"story.same", "text":"乙"}
  ]
}`)
	writeFixture(t, root, "remake/cmd/fd2/ui.go", "package main\n")
	if _, err := build(root); err == nil {
		t.Fatal("duplicate existing string_id was accepted")
	}
}

func TestMultipleTextFieldsDoNotReuseOneExistingStringID(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "remake/assets/story/ch00.json", `{
  "string_id":"story.line", "speaker_name":"索爾", "text":"出發"
}`)
	writeFixture(t, root, "remake/cmd/fd2/ui.go", "package main\n")
	inventory, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Entries) != 2 {
		t.Fatalf("entries=%d", len(inventory.Entries))
	}
	for _, entry := range inventory.Entries {
		if entry.StringID == "story.line" || entry.IDStatus == "existing_stable_id" {
			t.Fatalf("shared string_id was reused: %+v", entry)
		}
	}
}

func TestVariablesIncludeWrappedErrorPlaceholder(t *testing.T) {
	if got := variables("讀取失敗：%w"); !reflect.DeepEqual(got, []string{"%w"}) {
		t.Fatalf("variables=%v", got)
	}
}

func TestDrawFunctionContextIncludesIndirectASCIIAndFunctionName(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "remake/assets/story/ch00.json", `{"text":"測試"}`)
	writeFixture(t, root, "remake/cmd/fd2/ui.go", `package main
func drawTown() { label := "TOWN"; use(label) }
`)
	inventory, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range inventory.Entries {
		if entry.Text == "TOWN" {
			if entry.Role != "runtime_ui" || entry.Source.Function != "drawTown" {
				t.Fatalf("entry=%+v", entry)
			}
			return
		}
	}
	t.Fatal("indirect draw string was not collected")
}

func TestReviewedGoCandidatesMatchCurrentInventory(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", "..", ".."))
	inventory, err := build(repo)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))

	reviewRaw, err := os.ReadFile(filepath.Join(repo, "docs", "data", "fd2-string-review.json"))
	if err != nil {
		t.Fatal(err)
	}
	var review reviewDocument
	if err := json.Unmarshal(reviewRaw, &review); err != nil {
		t.Fatal(err)
	}
	if review.InventorySHA256 != digest || review.ReviewedRole != "go_review" {
		t.Fatalf("review binds sha=%s role=%s, want sha=%s role=go_review", review.InventorySHA256, review.ReviewedRole, digest)
	}

	want := map[string]bool{}
	for _, entry := range inventory.Entries {
		if entry.Role == review.ReviewedRole {
			want[entry.StringID] = true
		}
	}
	seen := map[string]string{}
	for disposition, group := range review.Dispositions {
		for _, id := range group.StringIDs {
			if previous, duplicate := seen[id]; duplicate {
				t.Fatalf("review id %s appears in %s and %s", id, previous, disposition)
			}
			if !want[id] {
				t.Fatalf("review id %s is not a current %s candidate", id, review.ReviewedRole)
			}
			seen[id] = disposition
		}
	}
	if len(seen) != len(want) {
		for id := range want {
			if seen[id] == "" {
				t.Errorf("unreviewed candidate %s", id)
			}
		}
	}
}
