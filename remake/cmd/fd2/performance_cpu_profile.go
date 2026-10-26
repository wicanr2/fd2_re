package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

type performanceCPUProfile struct {
	file     *os.File
	timer    *time.Timer
	finished sync.Once
	err      error
	done     chan struct{}
}

func startPerformanceCPUProfile(path string, duration time.Duration) (*performanceCPUProfile, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("CPU效能觀測輸出：%w", err)
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("CPU效能觀測啟動：%w", err)
	}
	profile := &performanceCPUProfile{file: file, done: make(chan struct{})}
	profile.timer = time.AfterFunc(duration, profile.finish)
	return profile, nil
}

func (p *performanceCPUProfile) finish() {
	p.finished.Do(func() {
		pprof.StopCPUProfile()
		p.err = p.file.Close()
		close(p.done)
	})
}

func (p *performanceCPUProfile) Close() error {
	p.timer.Stop()
	p.finish()
	return p.err
}
