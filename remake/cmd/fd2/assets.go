// assets.go — 資產路徑解析層(打包前置,doc38 §6.5)。
//
// 唯讀資產(assets/...)五層查找,先中先贏,不混層:
//  1. XDG 使用者覆蓋層 $XDG_DATA_HOME/fd2_re/assets/...
//     (doc38「使用者資料優先」原則:玩家編輯過的版本 / 版權衍生素材如 sprites·music·portraits·
//     maps 皆放這裡,AppImage 只打包 scenarios/story 等原創內容,故這些目錄實務上只在此層出現)
//  2. AppImage 唯讀基底 $APPDIR/assets/...(有設 APPDIR 才查;AppImage 執行期自動設)
//  3. macOS app bundle 的 Contents/Resources 相對
//  4. 執行檔所在目錄 相對(playfix #2:雙擊 / 從別的 cwd 用絕對路徑啟動 ./fd2-linux 時,
//     cwd 不一定是 remake/,資產解不到會靜默跳過開場動畫直接進戰場;開發模式 cwd=remake/
//     時這層與第4層同值,行為不變)
//  5. cwd 與祖先目錄(開發模式；供 `go test` 在 cmd/fd2 package cwd 回到 remake/)
//
// 可寫檔(存檔/設定)一律走 $XDG_DATA_HOME/fd2_re/,不再用 cwd(唯讀 mount 內無法寫入)。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/editorcanonical"
)

var exeDirCached string
var exeDirLooked bool

// exeDir 回傳執行檔所在目錄(符號連結已解),取不到則回傳 ""。
func exeDir() string {
	if exeDirLooked {
		return exeDirCached
	}
	exeDirLooked = true
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if rp, err := filepath.EvalSymlinks(p); err == nil {
		p = rp
	}
	exeDirCached = filepath.Dir(p)
	return exeDirCached
}

var userDataDirCached string

// userDataDir 回傳(並確保存在)可寫使用者資料夾:$XDG_DATA_HOME/fd2_re/,預設 ~/.local/share/fd2_re/。
func userDataDir() string {
	if userDataDirCached != "" {
		return userDataDirCached
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(base, "fd2_re")
	os.MkdirAll(dir, 0o755)
	userDataDirCached = dir
	return dir
}

// userDataPath 回傳可寫檔的完整路徑(存檔/設定/編輯器輸出)。
func userDataPath(name string) string {
	return filepath.Join(userDataDir(), name)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// macBundleResourceDir 回傳標準 .app bundle 的 Resources 目錄候選。
// 只辨識 Contents/MacOS 這個結構；實際資產仍須由呼叫端以存在性檢查承認。
func macBundleResourceDir() string {
	d := exeDir()
	if d == "" || filepath.Base(d) != "MacOS" {
		return ""
	}
	contents := filepath.Dir(d)
	if filepath.Base(contents) != "Contents" {
		return ""
	}
	return filepath.Join(contents, "Resources")
}

// assetPath 解析一個唯讀資產相對路徑(如 "assets/map.json" 或 "assets/maps/map3" 目錄)。
// 絕對路徑(系統字型等)原樣回傳。五層都沒有 → 回傳未改寫的 cwd 相對路徑,
// 呼叫端的 os.ReadFile 自然得到「檔案不存在」錯誤,行為與改動前一致。
func assetPath(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	if p := filepath.Join(userDataDir(), rel); fileExists(p) {
		return p
	}
	if appdir := os.Getenv("APPDIR"); appdir != "" {
		if p := filepath.Join(appdir, rel); fileExists(p) {
			return p
		}
	}
	if resources := macBundleResourceDir(); resources != "" {
		if p := filepath.Join(resources, rel); fileExists(p) {
			return p
		}
	}
	if d := exeDir(); d != "" {
		if p := filepath.Join(d, rel); fileExists(p) {
			return p
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for dir, i := cwd, 0; i < 5; dir, i = filepath.Dir(dir), i+1 {
			if p := filepath.Join(dir, rel); fileExists(p) {
				return p
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	return rel
}

// assetGlob 同 assetPath 的五層查找,用於萬用字元批次載入(sprite/portrait/figani)。
// 第一層有命中(非空)就整層採用,不同層的檔案不混拼。
func assetGlob(pattern string) []string {
	if m, _ := filepath.Glob(filepath.Join(userDataDir(), pattern)); len(m) > 0 {
		return m
	}
	if appdir := os.Getenv("APPDIR"); appdir != "" {
		if m, _ := filepath.Glob(filepath.Join(appdir, pattern)); len(m) > 0 {
			return m
		}
	}
	if resources := macBundleResourceDir(); resources != "" {
		if m, _ := filepath.Glob(filepath.Join(resources, pattern)); len(m) > 0 {
			return m
		}
	}
	if d := exeDir(); d != "" {
		if m, _ := filepath.Glob(filepath.Join(d, pattern)); len(m) > 0 {
			return m
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for dir, i := cwd, 0; i < 5; dir, i = filepath.Dir(dir), i+1 {
			if m, _ := filepath.Glob(filepath.Join(dir, pattern)); len(m) > 0 {
				return m
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	return nil
}

// packageSelfCheck 驗證正式封包必帶的可散布資料，也同時驗證 assetPath
// 能從封包結構解析所有劇情引用。它刻意不要求玩家自行提供的原版衍生資產。
func packageSelfCheck() error {
	if _, err := editorcanonical.ValidateBundle(assetPath("assets/editor-canonical")); err != nil {
		return fmt.Errorf("載入編輯器 canonical bundle: %w", err)
	}
	for _, localeID := range localeIDs {
		if _, err := loadOfficialLocale(localeID); err != nil {
			return fmt.Errorf("載入官方語系 %s: %w", localeID, err)
		}
		if _, err := loadOfficialLocaleContent(localeID); err != nil {
			return fmt.Errorf("載入官方全量內容 %s: %w", localeID, err)
		}
	}
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		return fmt.Errorf("載入完整戰役: %w", err)
	}
	spells, err := battle.LoadSpells(assetPath("assets/spells.json"))
	if err != nil {
		return fmt.Errorf("載入法術表: %w", err)
	}
	if len(spells) != 36 {
		return fmt.Errorf("法術表筆數為 %d，應為 36", len(spells))
	}
	seenSpell := [36]bool{}
	for _, spell := range spells {
		if spell.ID < 0 || spell.ID >= len(seenSpell) || seenSpell[spell.ID] {
			return fmt.Errorf("法術表含無效或重複 ID %d", spell.ID)
		}
		seenSpell[spell.ID] = true
	}

	scriptSet := make(map[string]struct{})
	for _, node := range graph.Nodes {
		if node.Script != "" {
			scriptSet[node.Script] = struct{}{}
		}
	}
	scripts := make([]string, 0, len(scriptSet))
	for rel := range scriptSet {
		scripts = append(scripts, rel)
	}
	sort.Strings(scripts)
	for _, rel := range scripts {
		path := assetPath(rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("讀取戰役引用 %s: %w", rel, err)
		}
		if !json.Valid(raw) {
			return fmt.Errorf("戰役引用 %s 不是有效 JSON", rel)
		}
	}
	return nil
}
