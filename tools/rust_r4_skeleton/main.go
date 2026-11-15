package main

// rust_r4_skeleton：R4 階段 1 的 fd2-game 骨架產生器（docs/goal/120）。
//
// 用法（工作目錄不限；由 tools/rust_r4_skeleton.sh 在 Docker 內呼叫）：
//
//	rust_r4_skeleton -deps <go list -export -deps -json 輸出> -repo <儲存庫根> -out <輸出目錄> [-stats]
//	rust_r4_skeleton -deps <...> -repo <...> -out <...> -r5 <executed-draw-functions.json>（R5 階段 1，見 r5.go）
//
// 輸入：remake/cmd/fd2 的正式程式（go list 選出的 GoFiles，不含 _test.go）、
// docs/data/rust-r4-ch01-coverage-20261114.json（階段 0 的 565 個函式），以及
// remake-rs 既有 crate 的 Rust 原始碼（外部型別名稱與 derive）。輸出只寫進 -out。

import (
	"flag"
	"path/filepath"
)

var (
	flagDeps  = flag.String("deps", "", "go list -export -deps -json ./cmd/fd2 的輸出檔")
	flagRepo  = flag.String("repo", "", "儲存庫根目錄")
	flagOut   = flag.String("out", "", "輸出目錄（crate 根）")
	flagStats = flag.Bool("stats", false, "只印統計")
	flagR5    = flag.String("r5", "", "R5 階段 1：只依這份 executed-draw-functions.json 輸出 r5-plan.json（不產生 crate）")
)

func main() {
	flag.Parse()
	rustScan = scanRust(*flagRepo)
	pd := loadPackage(*flagDeps)
	cov := loadCoverage(filepath.Join(*flagRepo, "docs/data/rust-r4-ch01-coverage-20261114.json"))
	s := newScope(pd, cov)
	s.printStats(*flagStats)
	if *flagR5 != "" {
		generateR5(s, *flagOut, *flagR5)
		return
	}
	if *flagStats {
		return
	}
	generate(s, *flagOut)
}
