// fd2-golden-figani 輸出 internal/figani 的黃金差分資料，供 Rust 改寫逐欄比對
// （docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 figani 的公開 API（以及 fdother 讀輸入），不改任何遊戲行為。
// 隨機輸入以固定種子的 PCG 產生，連同結果寫進同一份 JSON，Rust 端不需重現亂數產生器。
// 真實原版檔案只記錄路徑與 SHA-256，Rust 端以同一路徑重讀。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type caseJSON struct {
	Index  int            `json:"index"`
	Kind   string         `json:"kind"`
	Op     string         `json:"op"`
	Input  map[string]any `json:"input"`
	Result any            `json:"result"`
}

type exporter struct {
	rng   *rand.Rand
	cases []caseJSON
	// 合成的分離素材樹放在 work 之下，路徑在錯誤字串中以 $ROOT 取代；真實分離包的
	// animations 目錄以 $PACK 取代。Rust 端把同樣的佔位字換成自己的路徑。
	work string
	pack string
}

func (e *exporter) expand(root string) string {
	return strings.ReplaceAll(strings.ReplaceAll(root, "$PACK", e.pack), "$ROOT", e.work)
}

func (e *exporter) add(kind, op string, input map[string]any, result any) {
	e.cases = append(e.cases, caseJSON{Index: len(e.cases), Kind: kind, Op: op, Input: input, Result: result})
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func fileInfo(path string) map[string]any {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return map[string]any{"path": path, "size": len(raw), "sha256": digest(raw)}
}

// treeInfo 以排序過的相對路徑與各檔 SHA-256 摘要整棵目錄，不嵌入檔案內容。
func treeInfo(root string) map[string]any {
	var lines []string
	count := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		lines = append(lines, rel+" "+digest(raw))
		count++
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sort.Strings(lines)
	return map[string]any{"path": root, "files": count, "manifest_sha256": digest([]byte(strings.Join(lines, "\n")))}
}

func main() {
	out := flag.String("out", "", "黃金 JSON 輸出路徑")
	flame2 := flag.String("flame2", "", "原版 FLAME2 目錄（FIGANI.DAT、FDOTHER.DAT）")
	pack := flag.String("pack", "", "分離素材包根目錄（含 animations/）")
	entries := flag.String("entries", "", "remake/assets/data/native_command_indexed_entries.json")
	work := flag.String("work", "", "合成分離素材樹的工作目錄（必須是空目錄或不存在）")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flag.Parse()
	if *out == "" || *flame2 == "" || *pack == "" || *entries == "" || *work == "" {
		fmt.Fprintln(os.Stderr, "需要 -out -flame2 -pack -entries -work")
		os.Exit(2)
	}
	if err := os.RemoveAll(*work); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, *seed^0x9e3779b97f4a7c15)), work: *work}

	figaniPath := filepath.Join(*flame2, "FIGANI.DAT")
	fdotherPath := filepath.Join(*flame2, "FDOTHER.DAT")
	animations := filepath.Join(*pack, "animations")
	inputs := map[string]any{
		"figani_dat":      fileInfo(figaniPath),
		"fdother_dat":     fileInfo(fdotherPath),
		"pack_animations": treeInfo(animations),
		"indexed_entries": fileInfo(*entries),
	}

	e.realArchives(figaniPath, fdotherPath)
	e.realSeparated(animations)
	e.parseCases()
	e.schedulerCases()
	e.blitCases()
	e.nativeCases()
	e.strikeCases(figaniPath)
	e.indexedTableCases(*entries)
	e.separatedTreeCases()
	if err := os.RemoveAll(*work); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	kinds := map[string]int{}
	for _, c := range e.cases {
		kinds[c.Kind]++
	}
	doc := map[string]any{
		"schema": "fd2-golden-figani/1",
		"seed":   *seed,
		"inputs": inputs,
		"kinds":  kinds,
		"cases":  e.cases,
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("寫出 %d 案例：%v\n", len(e.cases), kinds)
}
