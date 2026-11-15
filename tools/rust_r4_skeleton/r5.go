package main

// R5 階段 1（docs/goal/121）：替階段 0 盤點到、開場與第一章執行到的非 HD 繪圖函式產生 R4 規則的簽名骨架。
//
// 用法：rust_r4_skeleton -deps <...> -repo <...> -out <...> -r5 <executed-draw-functions.json>
//
// 範圍與型別閉包照 R4 階段 1（docs/data/rust-r4-ch01-coverage-20261114.json）計算，不因這些函式擴大；骨架只
// 產生函式簽名，簽名用到閉包外的型別時直接失敗。輸出 <out>/r5-plan.json：盤點的每個函式一筆（檔名、Go 名稱、
// 接收者、行號、分組、是否 HD、Rust 路徑），非 HD 的另附 Rust 原始碼片段（doc＋簽名＋
// `todo!("r5:<檔>.go:<函式名>")`）。要不要插入 crate、PORTING.json 改成什麼狀態，由
// tools/rust_r5_skeleton_apply.py 依 crate 現況決定（已移植的不動）。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type r5Inventory struct {
	Schema    string `json:"schema"`
	Functions []struct {
		Func      string `json:"func"`
		File      string `json:"file"`
		Lines     [2]int `json:"lines"`
		LineCount int    `json:"line_count"`
		Group     string `json:"group"`
	} `json:"functions"`
}

type r5PlanEntry struct {
	File      string `json:"file"`
	Func      string `json:"func"` // 盤點的名稱，例如 (*Game).drawRing
	Name      string `json:"name"`
	Recv      string `json:"recv,omitempty"`
	Line      int    `json:"go_line"`
	EndLine   int    `json:"go_end_line"`
	LineCount int    `json:"line_count"`
	Group     string `json:"group"`
	HD        bool   `json:"hd"`
	Rust      string `json:"rust"`
	ImplType  string `json:"impl,omitempty"` // 方法所在的 Rust 型別
	Code      string `json:"code,omitempty"` // 非 HD：骨架原始碼（方法已縮排四格）
}

func generateR5(s *scope, out, invPath string) {
	raw, err := os.ReadFile(invPath)
	must(err)
	var inv r5Inventory
	must(json.Unmarshal(raw, &inv))
	if inv.Schema != "fd2_rust_r5_executed_draw_functions/1" {
		panic("盤點檔 schema 不對：" + inv.Schema)
	}
	g := newGen(s)
	a := newAnalyzer(s)
	computeKeys(s.pd)
	computeKeyTypes(g)
	computeVarWritten(g)
	g.computeTraits()
	computeNames(g, s)

	byLine := map[string]*fnInfo{}
	for _, fi := range s.fns {
		byLine[fmt.Sprintf("%s:%d", fi.file.Name, s.pd.line(fi.decl.Pos()))] = fi
	}
	var plan []r5PlanEntry
	for _, f := range inv.Functions {
		fi := byLine[fmt.Sprintf("%s:%d", f.File, f.Lines[0])]
		if fi == nil {
			panic("盤點的函式對不到 Go 宣告：" + f.File + ":" + f.Func)
		}
		e := r5PlanEntry{File: f.File, Func: f.Func, Name: fi.decl.Name.Name, Line: f.Lines[0], EndLine: f.Lines[1],
			LineCount: f.LineCount, Group: f.Group, HD: isHDFile(f.File) || strings.HasPrefix(f.Group, "HD"), Rust: rustPath(fi)}
		if fi.recvTN != nil {
			e.Recv = fi.recvTN.Name()
		}
		if !e.HD {
			if fi.sig.TypeParams().Len() > 0 {
				panic("泛型函式不在骨架範圍：" + fi.key)
			}
			if fi.recvTN != nil {
				rn, ok := g.typeNames[fi.recvTN]
				if !ok {
					panic("接收者型別不在 R4 型別閉包：" + fi.key)
				}
				e.ImplType = rn
			}
			// 只產生函式本身：先把 impl 視為已開啟，emitter 就不會輸出 impl 區塊。
			wasSel := fi.sel
			fi.sel = true
			nm.todo[fi] = "r5:" + nm.todo[fi]
			em := &emitter{g: g, a: a, s: s, pd: s.pd, f: fi.file, openImpl: e.ImplType}
			em.funcDecl(fi.decl)
			e.Code = strings.TrimPrefix(em.b.String(), "\n")
			fi.sel = wasSel
		}
		plan = append(plan, e)
	}
	sort.SliceStable(plan, func(i, j int) bool {
		if plan[i].File != plan[j].File {
			return plan[i].File < plan[j].File
		}
		return plan[i].Line < plan[j].Line
	})
	sum := sha256Hex(raw)
	data, err := json.MarshalIndent(map[string]any{
		"schema":    "fd2_rust_r5_skeleton_plan/1",
		"producer":  "tools/rust_r4_skeleton -r5（tools/rust_r5_skeleton.sh）",
		"inventory": map[string]string{"file": filepath.Base(invPath), "sha256": sum},
		"functions": plan,
	}, "", "  ")
	must(err)
	writeOut(out, "r5-plan.json", string(data)+"\n")
	nonHD := 0
	for _, e := range plan {
		if !e.HD {
			nonHD++
		}
	}
	fmt.Fprintf(os.Stderr, "STAT r5 inventory=%d non_hd=%d hd=%d\n", len(plan), nonHD, len(plan)-nonHD)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
