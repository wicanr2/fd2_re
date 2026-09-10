//go:build js

package main

import "path"

// separatedAssetPackPrefix 是網頁版取用分離素材的路徑前綴。
//
// 桌面版用 FD2_ASSET_PACK 把原版衍生資產與公開庫資產分成兩個命名空間；瀏覽器沒有
// 環境變數，所以改用固定前綴。這件事不能省：兩邊各有 fonts／maps／music／
// portraits／sfx／sprites／ui 七個同名目錄，混在同一個路徑底下會互相蓋掉。
const separatedAssetPackPrefix = "pack"

func separatedAssetPath(relative string) string {
	return path.Join(separatedAssetPackPrefix, relative)
}
