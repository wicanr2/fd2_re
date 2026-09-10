//go:build !js

package main

import (
	"os"
	"path/filepath"
)

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
