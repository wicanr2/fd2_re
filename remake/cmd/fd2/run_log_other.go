//go:build !windows

package main

import "os"

// 其他平台的 stderr 仍可由終端機或啟動器取得；明示 FD2_RUN_LOG=1 才另存紀錄檔。
// 這時只改 Go 的 os.Stderr 與 log，runtime 崩潰堆疊仍寫原本的 fd 2。
func runLogWanted() bool { return os.Getenv("FD2_RUN_LOG") == "1" }

func redirectProcessStderr(*os.File) {}
