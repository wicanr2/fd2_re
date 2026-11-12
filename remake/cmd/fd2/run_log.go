// run_log.go — 玩家版執行紀錄。
//
// Windows 正式版以 -H=windowsgui 建置，沒有主控台；程式崩潰時 Go runtime 寫到
// stderr 的 panic 與 fatal error 會整段遺失。這裡在啟動時把 stderr 與標準 log
// 導向使用者資料夾的 fd2-run.log（前一次執行保留為 fd2-run.prev.log），並在
// 每個 Update 尾端記錄 loadErr／狀態訊息的變化，讓玩家回報時能附上當時的錯誤。
package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
)

const (
	runLogName     = "fd2-run.log"
	runLogPrevName = "fd2-run.prev.log"
)

var runLogFile *os.File

// openRunLog 只在平台需要時啟用（見 runLogWanted）。失敗時不影響遊戲啟動。
func openRunLog() {
	if !runLogWanted() || runLogFile != nil {
		return
	}
	path := userDataPath(runLogName)
	if fileExists(path) {
		_ = os.Rename(path, userDataPath(runLogPrevName))
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}
	runLogFile = f
	redirectProcessStderr(f)
	os.Stderr = f
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	fmt.Fprintf(f, "FD2 %s %s/%s %s start %s\n", buildVersion, runtime.GOOS, runtime.GOARCH,
		runtime.Version(), time.Now().Format(time.RFC3339))
}

// runLogTrace 記錄玩家看得到的錯誤與狀態訊息轉換；同一訊息不重複寫。
type runLogTrace struct {
	loadErr, msg string
}

func (g *Game) traceRunLog() {
	if runLogFile == nil || g == nil {
		return
	}
	if g.loadErr != g.runLog.loadErr {
		g.runLog.loadErr = g.loadErr
		if g.loadErr != "" {
			log.Printf("frame=%d node=%s loadErr=%q", g.frame, g.runLogNode(), g.loadErr)
		}
	}
	if g.msg != g.runLog.msg {
		g.runLog.msg = g.msg
		if g.msg != "" {
			log.Printf("frame=%d node=%s msg=%q", g.frame, g.runLogNode(), g.msg)
		}
	}
}

func (g *Game) runLogNode() string {
	if g.camp == nil {
		return "-"
	}
	return g.camp.Cur
}
