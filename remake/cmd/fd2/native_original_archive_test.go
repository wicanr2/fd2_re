package main

import "os"

// 原版 archive locator 只供 source-oracle 測試；正式遊戲 binary 不得編入
// FD2_ORIGINAL_* 或 assets/original/*.DAT 回退入口。
func nativeOriginalArchivePath(environment, name string) string {
	if path := os.Getenv(environment); path != "" {
		return path
	}
	path := assetPath("assets/original/" + name)
	if fileExists(path) {
		return path
	}
	return ""
}

func nativeFDOTHERPath() string {
	return nativeOriginalArchivePath("FD2_ORIGINAL_FDOTHER", "FDOTHER.DAT")
}

func nativeDATOPath() string {
	return nativeOriginalArchivePath("FD2_ORIGINAL_DATO", "DATO.DAT")
}
