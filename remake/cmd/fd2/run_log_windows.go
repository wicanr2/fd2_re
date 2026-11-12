//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// windowsgui 程式沒有主控台，stderr 一律導向紀錄檔。
func runLogWanted() bool { return os.Getenv("FD2_RUN_LOG") != "0" }

// Go runtime 在 Windows 每次寫 fd 2 都重新呼叫 GetStdHandle(STD_ERROR_HANDLE)
// （go1.22 runtime/os_windows.go write1），所以換掉行程的標準錯誤 handle 後，
// panic 與 fatal error 的堆疊也會寫進紀錄檔。
func redirectProcessStderr(f *os.File) {
	_ = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd()))
}
