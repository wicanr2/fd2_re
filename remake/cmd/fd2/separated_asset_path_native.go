//go:build !js

package main

import (
	"os"
	"path/filepath"
)

// separatedAssetPath 解析分離素材（原版衍生資產）。FD2_ASSET_PACK 指定素材包時
// 一律以它為準；沒有指定就退回一般資產查找，讓「素材放在遊戲目錄旁」的安裝方式
// 也能運作。
func separatedAssetPath(relative string) string {
	if root := os.Getenv("FD2_ASSET_PACK"); root != "" {
		return filepath.Join(root, filepath.FromSlash(relative))
	}
	return assetPath(filepath.ToSlash(filepath.Join("assets", relative)))
}
