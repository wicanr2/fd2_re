//go:build js

package main

import (
	"encoding/json"
	"os"
	"path"
	"sync"
)

// assets_glob_js.go — 網頁版的 assetGlob。
//
// HTTP 沒有「列目錄」這回事，js/wasm 底下也沒有檔案系統可掃，所以萬用字元查找
// 改成對一份索引做比對。索引由 tools/build_asset_manifest.py 產生，內容是
// `assetGlob` 實際用到的那幾個目錄的完整檔案清單。
//
// 索引載不到時回空（而不是猜），呼叫端本來就把「找不到成對資源」當成失敗即關閉。

var (
	assetManifestOnce  sync.Once
	assetManifestFiles []string
	assetManifestErr   error
)

// loadAssetManifest 讀一次索引。os.ReadFile 在網頁版由 index.html 的檔案系統
// 轉接層服務，見那裡的說明。
func loadAssetManifest() []string {
	assetManifestOnce.Do(func() {
		raw, err := os.ReadFile("assets-manifest.json")
		if err != nil {
			assetManifestErr = err
			return
		}
		var doc struct {
			Files []string `json:"files"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			assetManifestErr = err
			return
		}
		assetManifestFiles = doc.Files
	})
	return assetManifestFiles
}

// assetGlob 用索引重現 filepath.Glob 的語意：`*` 不跨過路徑分隔線，順序照索引
// （產生器已經排序過），所以同一份索引每次得到同一個結果。
func assetGlob(pattern string) []string {
	var out []string
	for _, name := range loadAssetManifest() {
		if ok, err := path.Match(pattern, name); ok && err == nil {
			out = append(out, name)
		}
	}
	return out
}
