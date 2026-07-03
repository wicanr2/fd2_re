// assets.go — 資產路徑解析層(打包前置,doc38 §6.5)。
//
// 唯讀資產(assets/...)四層查找,先中先贏,不混層:
//  1. XDG 使用者覆蓋層 $XDG_DATA_HOME/fd2_re/assets/...
//     (doc38「使用者資料優先」原則:玩家編輯過的版本 / 版權衍生素材如 sprites·music·portraits·
//     maps 皆放這裡,AppImage 只打包 scenarios/story 等原創內容,故這些目錄實務上只在此層出現)
//  2. AppImage 唯讀基底 $APPDIR/assets/...(有設 APPDIR 才查;AppImage 執行期自動設)
//  3. 執行檔所在目錄 相對(playfix #2:雙擊 / 從別的 cwd 用絕對路徑啟動 ./fd2-linux 時,
//     cwd 不一定是 remake/,資產解不到會靜默跳過開場動畫直接進戰場;開發模式 cwd=remake/
//     時這層與第4層同值,行為不變)
//  4. cwd 相對(開發模式既有行為,未設 APPDIR、也無 XDG 覆蓋時)
//
// 可寫檔(存檔/設定)一律走 $XDG_DATA_HOME/fd2_re/,不再用 cwd(唯讀 mount 內無法寫入)。
package main

import (
	"os"
	"path/filepath"
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

// assetPath 解析一個唯讀資產相對路徑(如 "assets/map.json" 或 "assets/maps/map3" 目錄)。
// 絕對路徑(系統字型等)原樣回傳。三層都沒有 → 回傳未改寫的 cwd 相對路徑,
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
	if d := exeDir(); d != "" {
		if p := filepath.Join(d, rel); fileExists(p) {
			return p
		}
	}
	return rel
}

// assetGlob 同 assetPath 的三層查找,但用於萬用字元批次載入(sprite/portrait/figani)。
// 依序試三層,第一層有命中(非空)就整層採用,不同層的檔案不混拼。
func assetGlob(pattern string) []string {
	if m, _ := filepath.Glob(filepath.Join(userDataDir(), pattern)); len(m) > 0 {
		return m
	}
	if appdir := os.Getenv("APPDIR"); appdir != "" {
		if m, _ := filepath.Glob(filepath.Join(appdir, pattern)); len(m) > 0 {
			return m
		}
	}
	if d := exeDir(); d != "" {
		if m, _ := filepath.Glob(filepath.Join(d, pattern)); len(m) > 0 {
			return m
		}
	}
	m, _ := filepath.Glob(pattern)
	return m
}
