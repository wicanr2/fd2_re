// fd2-golden-gorand 輸出 Go math/rand 在 battle 用到的取樣序列，供 Rust 版 battle/rng.rs 逐值比對
// （docs/goal/119-goal-rust-bevy-rewrite-20261113.md、remake-rs/crates/fd2-rules/PORTING.md
// 「開工前的決定」第 1 點）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫標準函式庫 rand.New(rand.NewSource(seed)) 的
// Int63／Intn，不碰任何遊戲程式。輸出完全由下列固定清單決定，重跑結果逐位元相同：
//   - mixed：每個種子一個 *rand.Rand，先取 4 個 Int63，再依 ns 清單輪流呼叫 Intn 共 len(ns)*4 次，
//     驗證不同 n 交錯時共用同一個來源。
//   - per_n：較少的種子，每個 (seed, n) 各自新建 *rand.Rand，連續取 32 個 Intn。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
)

type mixedCase struct {
	Seed  int64   `json:"seed"`
	Int63 []int64 `json:"int63"`
	Intn  []int64 `json:"intn"`
}

type perNCase struct {
	Seed   int64   `json:"seed"`
	N      int64   `json:"n"`
	Values []int64 `json:"values"`
}

// ns 涵蓋 1、2 的冪、非 2 的冪、int32 上限附近與 1<<31 以上（走 Int63n）。
var ns = []int64{
	1, 2, 3, 4, 5, 6, 7, 10, 12, 16, 20, 32, 64, 100, 255, 256, 1000, 1 << 16, 100000,
	1<<30 - 1, 1 << 30, 1<<31 - 2, 1<<31 - 1, 1 << 31, 1<<31 + 1, 3 << 31, 1 << 40, 1e12 + 7,
	1 << 62, 3 << 61, math.MaxInt64 - 1, math.MaxInt64,
}

func seeds() []int64 {
	out := make([]int64, 0, 1100)
	for seed := int64(0); seed < 1000; seed++ { // 測試的 seed 迴圈與常見小種子
		out = append(out, seed)
	}
	return append(out,
		-1, -2, -7, -42, -1000, -12345, -2147483647, -2147483648, -2147483649,
		1<<31-1, 1<<31, 2*(1<<31-1), 1<<31+5, 20260915, 20261113, 20261114,
		1700000000000000000, 1763078400123456789, math.MaxInt64, math.MinInt64, math.MinInt64+1,
		-(1 << 40), 1<<40, 89482311,
	)
}

// perNSeeds 是 per_n 用的較小種子集合（含負數、0 與 seed%(2^31-1)==0 的特例）。
var perNSeeds = []int64{0, 1, 2, 7, 42, 99, 123, 999, -1, -12345, 1<<31 - 1, math.MinInt64, math.MaxInt64, 20261114}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	var mixed []mixedCase
	for _, seed := range seeds() {
		r := rand.New(rand.NewSource(seed))
		c := mixedCase{Seed: seed}
		for i := 0; i < 4; i++ {
			c.Int63 = append(c.Int63, r.Int63())
		}
		for i := 0; i < 4*len(ns); i++ {
			c.Intn = append(c.Intn, int64(r.Intn(int(ns[i%len(ns)]))))
		}
		mixed = append(mixed, c)
	}
	var perN []perNCase
	for _, seed := range perNSeeds {
		for _, n := range ns {
			r := rand.New(rand.NewSource(seed))
			c := perNCase{Seed: seed, N: n}
			for i := 0; i < 32; i++ {
				c.Values = append(c.Values, int64(r.Intn(int(n))))
			}
			perN = append(perN, c)
		}
	}
	doc := map[string]any{
		"schema": "fd2-golden-gorand/1",
		"go":     runtime.Version(),
		"ns":     ns,
		"mixed":  mixed,
		"per_n":  perN,
	}
	data, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("mixed=%d per_n=%d\n", len(mixed), len(perN))
}
