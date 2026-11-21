package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/metrics"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// 以真實 HD 包量測主題載入（#317、#318）。只在 FD2_HD_LOAD_MEASURE=1 時執行，每個測試用
// 獨立程序跑（go test -run '^TestHDThemeLoadMeasureX$'），才能讓 VmHWM 只反映這一項。
// 這裡只放不經遊戲迴圈的載入器量測與共用工具；F2／F11／啟動在 hd_theme_load_gui_measure_test.go
// 以 ebiten.RunGame 量，離屏呼叫 Update／Draw 的迴圈沒有幀結束，heap 會隨幀數膨脹。
// 需要 FD2_ASSET_PACK、FD2_MODERN_THEME_PACK 指向完整素材；結果寫成 JSON 到
// FD2_HD_LOAD_MEASURE_OUT 目錄。

type hdLoadHeapSampler struct {
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	peakHeap uint64 // /memory/classes/heap/objects:bytes 的最大值
	peakAt   time.Duration
	start    time.Time
}

func startHDLoadHeapSampler() *hdLoadHeapSampler {
	s := &hdLoadHeapSampler{stop: make(chan struct{}), done: make(chan struct{}), start: time.Now()}
	samples := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			metrics.Read(samples)
			if v := samples[0].Value.Uint64(); v > s.peakHeap {
				s.mu.Lock()
				s.peakHeap, s.peakAt = v, time.Since(s.start)
				s.mu.Unlock()
			}
			select {
			case <-s.stop:
				return
			case <-ticker.C:
			}
		}
	}()
	return s
}

func (s *hdLoadHeapSampler) finish() (uint64, time.Duration) {
	close(s.stop)
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.peakHeap, s.peakAt
}

// hdLoadProcessStatus 讀 /proc/self/status 的 VmHWM／VmRSS（kB）。
func hdLoadProcessStatus() map[string]int64 {
	out := map[string]int64{}
	raw, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(raw), "\n") {
		for _, key := range []string{"VmHWM", "VmRSS"} {
			if strings.HasPrefix(line, key+":") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if n, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						out[key+"_kB"] = n
					}
				}
			}
		}
	}
	return out
}

func hdLoadLiveHeap() uint64 {
	runtime.GC()
	runtime.GC()
	samples := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
	metrics.Read(samples)
	return samples[0].Value.Uint64()
}

// hdLoadRetainedBytes 依 modernStoryPortraitSet 的頂層欄位統計留在記憶體的 []byte 容量
// （image 的 Pix、壓縮 PNG、音訊等）。同一底層陣列只算一次；不進入 ebiten 型別。
func hdLoadRetainedBytes(set *modernStoryPortraitSet) map[string]int64 {
	out := map[string]int64{}
	if set == nil {
		return out
	}
	seenPtr := map[uintptr]bool{}
	seenBytes := map[uintptr]bool{}
	var walk func(v reflect.Value) int64
	walk = func(v reflect.Value) int64 {
		if !v.IsValid() {
			return 0
		}
		if pkg := v.Type().PkgPath(); strings.Contains(pkg, "hajimehoshi/ebiten") {
			return 0
		}
		switch v.Kind() {
		case reflect.Pointer:
			if v.IsNil() {
				return 0
			}
			if strings.Contains(v.Type().Elem().PkgPath(), "hajimehoshi/ebiten") {
				return 0
			}
			p := v.Pointer()
			if seenPtr[p] {
				return 0
			}
			seenPtr[p] = true
			return walk(v.Elem())
		case reflect.Interface:
			if v.IsNil() {
				return 0
			}
			return walk(v.Elem())
		case reflect.Struct:
			var n int64
			for i := 0; i < v.NumField(); i++ {
				n += walk(v.Field(i))
			}
			return n
		case reflect.Slice:
			if v.IsNil() {
				return 0
			}
			if v.Type().Elem().Kind() == reflect.Uint8 {
				p := v.Pointer()
				if v.Cap() == 0 || seenBytes[p] {
					return 0
				}
				seenBytes[p] = true
				return int64(v.Cap())
			}
			var n int64
			for i := 0; i < v.Len(); i++ {
				n += walk(v.Index(i))
			}
			return n
		case reflect.Array:
			var n int64
			for i := 0; i < v.Len(); i++ {
				n += walk(v.Index(i))
			}
			return n
		case reflect.Map:
			var n int64
			iter := v.MapRange()
			for iter.Next() {
				n += walk(iter.Value())
			}
			return n
		}
		return 0
	}
	root := reflect.ValueOf(set).Elem()
	for i := 0; i < root.NumField(); i++ {
		if n := walk(root.Field(i)); n != 0 {
			out[root.Type().Field(i).Name] = n
		}
	}
	return out
}

// hdLoadSetDigestSkip 是只記錄延後檢查或按需解碼狀態的欄位，不屬於載入內容，比對修正前後時略過。
// 「型別.欄位」只略過該型別的欄位：地形圖的 PNG 路徑與雜湊是按需解碼的來源（#317 後續）。
var hdLoadSetDigestSkip = map[string]bool{"coverage": true, "verified": true, "notices": true,
	"modernTerrainHDAsset.path": true, "modernTerrainHDAsset.hash": true}

// hdLoadSetDigest 以固定順序（map 依鍵排序）把整個主題集合的內容雜湊成一個值：所有純量、
// 字串與 []byte（含圖片像素與壓縮 PNG）。修正前後載入同一份包，這個值相同即內容相同。
func hdLoadSetDigest(set *modernStoryPortraitSet) string {
	return hdLoadValueDigest(reflect.ValueOf(set).Elem())
}

// hdLoadValueDigest 是 hdLoadSetDigest 的走訪，可用在集合的單一欄位。
func hdLoadValueDigest(root reflect.Value) string {
	h := sha256.New()
	seen := map[uintptr]bool{}
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		if !v.IsValid() {
			fmt.Fprint(h, "<invalid>")
			return
		}
		if strings.Contains(v.Type().PkgPath(), "hajimehoshi/ebiten") {
			return
		}
		switch v.Kind() {
		case reflect.Pointer:
			if v.IsNil() {
				fmt.Fprint(h, "nil;")
				return
			}
			if strings.Contains(v.Type().Elem().PkgPath(), "hajimehoshi/ebiten") || v.Type().Elem().PkgPath() == "sync" {
				return
			}
			if seen[v.Pointer()] {
				fmt.Fprint(h, "seen;")
				return
			}
			seen[v.Pointer()] = true
			walk(v.Elem())
		case reflect.Interface:
			if v.IsNil() {
				fmt.Fprint(h, "nil;")
				return
			}
			fmt.Fprint(h, v.Elem().Type().String(), ";")
			walk(v.Elem())
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if hdLoadSetDigestSkip[v.Type().Field(i).Name] || hdLoadSetDigestSkip[v.Type().Name()+"."+v.Type().Field(i).Name] {
					continue
				}
				fmt.Fprint(h, v.Type().Field(i).Name, "=")
				walk(v.Field(i))
			}
		case reflect.Slice, reflect.Array:
			if v.Kind() == reflect.Slice && v.IsNil() {
				fmt.Fprint(h, "nil;")
				return
			}
			if v.Type().Elem().Kind() == reflect.Uint8 {
				fmt.Fprint(h, v.Len(), ":")
				if v.Kind() == reflect.Slice || v.CanAddr() {
					h.Write(v.Bytes())
					return
				}
				for i := 0; i < v.Len(); i++ {
					h.Write([]byte{byte(v.Index(i).Uint())})
				}
				return
			}
			fmt.Fprint(h, "[", v.Len(), "]")
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		case reflect.Map:
			keys := v.MapKeys()
			sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
			fmt.Fprint(h, "{", len(keys), "}")
			for _, k := range keys {
				fmt.Fprint(h, fmt.Sprint(k), ":")
				walk(v.MapIndex(k))
			}
		case reflect.Func, reflect.Chan, reflect.UnsafePointer:
			fmt.Fprint(h, v.Kind().String(), ";")
		default:
			fmt.Fprint(h, fmt.Sprint(v), ";")
		}
	}
	walk(root)
	return hex.EncodeToString(h.Sum(nil))
}

func hdLoadMeasureSetup(t *testing.T) string {
	t.Helper()
	if os.Getenv("FD2_HD_LOAD_MEASURE") != "1" {
		t.Skip("FD2_HD_LOAD_MEASURE")
	}
	out := os.Getenv("FD2_HD_LOAD_MEASURE_OUT")
	if out == "" {
		t.Fatal("FD2_HD_LOAD_MEASURE_OUT")
	}
	if os.Getenv("FD2_MODERN_THEME_PACK") == "" {
		t.Fatal("FD2_MODERN_THEME_PACK")
	}
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	return out
}

// hdLoadWriteHeapProfile 在 FD2_HD_LOAD_MEASURE_HEAP=1 時寫出切換後的 heap 剖析（inuse）。
func hdLoadWriteHeapProfile(t *testing.T, dir, name string) {
	t.Helper()
	if os.Getenv("FD2_HD_LOAD_MEASURE_HEAP") != "1" {
		return
	}
	f, err := os.Create(filepath.Join(dir, name+"-heap.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		t.Fatal(err)
	}
}

func hdLoadWriteReceipt(t *testing.T, dir, name string, receipt map[string]any) {
	t.Helper()
	receipt["go"] = runtime.Version()
	receipt["gomaxprocs"] = runtime.GOMAXPROCS(0)
	receipt["process"] = hdLoadProcessStatus()
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: %s", name, raw)
}

// TestHDThemeLoadMeasureLoader 直接呼叫正式載入函式：總時間、heap 高峰與載入後留存。
func TestHDThemeLoadMeasureLoader(t *testing.T) {
	out := hdLoadMeasureSetup(t)
	base := hdLoadLiveHeap()
	sampler := startHDLoadHeapSampler()
	start := time.Now()
	set, err := loadPlayerModernTheme()
	elapsed := time.Since(start)
	peak, peakAt := sampler.finish()
	if err != nil {
		t.Fatal(err)
	}
	live := hdLoadLiveHeap()
	retained := hdLoadRetainedBytes(set)
	keys := make([]string, 0, len(retained))
	var total int64
	for k, v := range retained {
		keys = append(keys, k)
		total += v
	}
	sort.Slice(keys, func(i, j int) bool { return retained[keys[i]] > retained[keys[j]] })
	ordered := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		ordered = append(ordered, map[string]any{"field": k, "bytes": retained[k]})
	}
	digest := hdLoadSetDigest(set)
	runtime.KeepAlive(set)
	hdLoadWriteReceipt(t, out, "loader", map[string]any{
		"set_digest":             digest,
		"load_seconds":           elapsed.Seconds(),
		"heap_objects_base":      base,
		"heap_objects_peak":      peak,
		"heap_objects_peak_at_s": peakAt.Seconds(),
		"heap_objects_live":      live,
		"retained_bytes_total":   total,
		"retained_bytes":         ordered,
	})
}
