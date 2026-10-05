//go:build ignore

// 在 Docker 的臨時 module 中以 go run main.go 執行。這是 #102 的原始
// 配置器局部診斷，不是章 oracle；人工 free list 與自然啟動分開記錄。
package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
)

const fixedEXE = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
const heapHeader, firstHole, roverHole, lastHole = uint32(0x180000), uint32(0x180100), uint32(0x180500), uint32(0x180900)
const labStack, labReturn = uint32(0x170000), uint32(0x90000)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func digest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func put(m *machine.LEMachine, at, value uint32) {
	if uint64(at)+4 > uint64(len(m.Mem)) {
		panic("lab write outside backed memory")
	}
	binary.LittleEndian.PutUint32(m.Mem[at:at+4], value)
}

func word(m *machine.LEMachine, at uint32) uint32 {
	v, err := m.Read32(at)
	must(err)
	return v
}

func load(data []byte) *machine.LEMachine {
	m, err := machine.LoadLE(data)
	must(err)
	if len(m.Mem) >= 0x170000 {
		panic("loaded image overlaps artificial lab")
	}
	m.Mem = append(m.Mem, make([]byte, 0x200000-len(m.Mem))...)
	return m
}

func originalAllocation(data []byte, name string, size, prefixMax uint32) map[string]any {
	m := load(data)
	sentinel := heapHeader + 0x1c
	put(m, heapHeader+8, roverHole)
	put(m, heapHeader+0xc, prefixMax)
	put(m, heapHeader+0x10, 128)
	put(m, heapHeader+0x14, 5)
	put(m, heapHeader+0x18, 3)
	put(m, sentinel+4, lastHole)
	put(m, sentinel+8, firstHole)
	holes := []uint32{firstHole, roverHole, lastHole}
	for i, at := range holes {
		for j := uint32(0); j < 128; j++ {
			m.Mem[at+j] = 0xa5
		}
		prev, next := sentinel, sentinel
		if i > 0 {
			prev = holes[i-1]
		}
		if i+1 < len(holes) {
			next = holes[i+1]
		}
		put(m, at, 128)
		put(m, at+4, prev)
		put(m, at+8, next)
	}
	before := append([]byte(nil), m.Mem[heapHeader:lastHole+128]...)
	put(m, labStack, labReturn)
	c := m.CPU
	c.EIP, c.R[cpu386.ESP] = 0x3d270, labStack
	c.R[cpu386.EAX], c.R[cpu386.EBX], c.R[cpu386.EDX] = size, heapHeader, 0x10
	var trace []uint32
	var stopped string
	for c.EIP != labReturn && len(trace) < 512 {
		trace = append(trace, c.EIP)
		if err := c.Step(); err != nil {
			stopped = err.Error()
			break
		}
	}
	if c.StepHook != nil {
		panic("original allocator hook unexpectedly installed")
	}
	if c.EIP != labReturn {
		return map[string]any{"name": name, "classification": "direct-entry artificial heap fixture; not normal player path",
			"status": "stopped", "error": stopped, "trace_eip": trace, "eip_after_step": c.EIP, "request": size,
			"input_rover": roverHole, "input_prefix_max": prefixMax}
	}
	if c.R[cpu386.ESP] != labStack+4 {
		panic("original allocator returned with invalid stack")
	}
	result := c.R[cpu386.EAX]
	var payload string
	var tag uint32
	if result != 0 {
		payload = fmt.Sprintf("%x", m.Mem[result:result+8])
		tag = word(m, result-4)
	}
	return map[string]any{"name": name, "status": "returned", "classification": "direct-entry artificial heap fixture; not normal player path",
		"request": size, "input_ebx": heapHeader, "input_edx": 0x10, "input_rover": roverHole,
		"input_prefix_max": prefixMax, "input_holes": holes, "result": result, "allocated_tag": tag,
		"result_first8_hex": payload, "rover_after": word(m, heapHeader+8), "prefix_max_after": word(m, heapHeader+0xc),
		"before_sha256": digest(before), "after_sha256": digest(m.Mem[heapHeader : lastHole+128]), "trace_eip": trace,
		"address_space": "dosgolem relocated LE linear; injected lab 0x170000..0x200000"}
}

func currentHook(data []byte) map[string]any {
	m := load(data)
	_, err := machine.InstallFD2WatcomRuntimeWithHeapCapacity(m, 32*1024*1024)
	must(err)
	call := func(entry, arg uint32) uint32 {
		put(m, labStack, labReturn)
		put(m, labStack+4, arg)
		m.CPU.EIP, m.CPU.R[cpu386.ESP] = entry, labStack
		must(m.CPU.Step())
		if m.CPU.EIP != labReturn || m.CPU.R[cpu386.ESP] != labStack+4 {
			panic("current hook did not return")
		}
		return m.CPU.R[cpu386.EAX]
	}
	var holes []uint32
	for i := 0; i < 3; i++ {
		at := call(0x36d26, 128)
		holes = append(holes, at)
		for j := uint32(0); j < 128; j++ {
			m.Mem[at+j] = 0xa5
		}
		if i < 2 {
			call(0x36d26, 896)
		}
	}
	for _, at := range holes {
		call(0x37426, at)
	}
	result := call(0x36d26, 24)
	if result != holes[0] || !bytes.Equal(m.Mem[result:result+24], make([]byte, 24)) {
		panic("current runtime policy changed; re-audit before comparing")
	}
	return map[string]any{"classification": "current runtime approximation; not native allocator", "request": 24,
		"holes": holes, "result": result, "result_first8_hex": fmt.Sprintf("%x", m.Mem[result:result+8]),
		"policy": "lowest-address first-fit; reused payload cleared; no native block header"}
}

func naturalFirstAllocation(data []byte) map[string]any {
	m, err := machine.LoadLE(data)
	must(err)
	initialSegments := m.CPU.Seg
	services := machine.NewFD2StartupDOS(nil)
	defer services.Close()
	m.CPU.IntHook = services.Handle
	_, err = machine.InstallFD2WatcomRuntimeWithHeapCapacity(m, 32*1024*1024, services.DPMI)
	must(err)
	installed := m.CPU.StepHook
	// 只在此診斷撤掉近堆與 argv 的合成轉接，保留原始 LE entry。
	// 沒有初始化人工 heap；其他已登記 DOS runtime 服務仍照工具契約。
	m.CPU.StepHook = func(c *cpu386.CPU) (bool, error) {
		if c.EIP == 0x36d26 || c.EIP == 0x37426 || c.EIP == 0x46114 {
			return false, nil
		}
		return installed(c)
	}
	var events []map[string]any
	var savedGSLinear uint32
	var savedGSValid bool
	snapshot := func() map[string]any {
		c := m.CPU
		state := map[string]any{"eip": c.EIP, "registers": c.R, "segments": c.Seg, "flags": c.EFlags}
		if desc, ok := c.Descriptors[c.Seg[cpu386.SegSS]]; ok {
			at := uint64(desc.Base) + uint64(c.R[cpu386.ESP])
			state["ss_descriptor"] = desc
			state["stack_linear"] = at
			if at+32 <= uint64(len(m.Mem)) {
				state["stack_next32_hex"] = fmt.Sprintf("%x", m.Mem[at:at+32])
			}
		}
		if savedGSValid {
			state["saved_gs_linear"] = savedGSLinear
			state["saved_gs_dword"] = binary.LittleEndian.Uint32(m.Mem[savedGSLinear : savedGSLinear+4])
		}
		return state
	}
	entryState := snapshot()
	steps := 0
	var tail []uint32
	var stopped string
	for steps < 100000 && m.CPU.EIP != 0x4cc51 {
		tail = append(tail, m.CPU.EIP)
		if len(tail) > 32 {
			tail = tail[len(tail)-32:]
		}
		before := snapshot()
		oldSeg, at := m.CPU.Seg, m.CPU.EIP
		raw := ""
		if uint64(at)+8 <= uint64(len(m.Mem)) {
			raw = fmt.Sprintf("%x", m.Mem[at:at+8])
		}
		err := m.CPU.Step()
		if err == nil && at == 0x36d2b {
			if desc, ok := m.CPU.Descriptors[m.CPU.Seg[cpu386.SegSS]]; ok {
				linear := uint64(desc.Base) + uint64(m.CPU.R[cpu386.ESP])
				if linear+4 <= uint64(len(m.Mem)) {
					savedGSLinear, savedGSValid = uint32(linear), true
				}
			}
		}
		after := snapshot()
		changedSavedGS := savedGSValid && before["saved_gs_dword"] != after["saved_gs_dword"]
		if err != nil || oldSeg != m.CPU.Seg || changedSavedGS || at == 0x36d26 || at == 0x36d2b || at == 0x36d90 {
			event := map[string]any{"steps_before": steps, "instruction_eip": at, "next8_hex": raw, "before": before, "after": after}
			if err != nil {
				event["error"] = err.Error()
			}
			events = append(events, event)
		}
		if err != nil {
			stopped = err.Error()
			break
		}
		steps++
	}
	status := "stopped"
	var stoppedEIP uint32
	var stoppedBytes string
	if stopped != "" && len(tail) > 0 {
		stoppedEIP = tail[len(tail)-1]
		if uint64(stoppedEIP)+8 <= uint64(len(m.Mem)) {
			stoppedBytes = fmt.Sprintf("%x", m.Mem[stoppedEIP:stoppedEIP+8])
		}
	}
	if m.CPU.EIP == 0x4cc51 {
		status = "first_caller_returned"
	}
	returnedState := snapshot()
	pointerCheck := map[string]any{"attempted": false}
	if status == "first_caller_returned" {
		ptr := m.CPU.R[cpu386.EAX]
		pointerCheck["attempted"] = true
		pointerCheck["request"] = 1
		pointerCheck["pointer_nonzero"] = ptr != 0
		if desc, ok := m.CPU.Descriptors[m.CPU.Seg[cpu386.SegDS]]; ok {
			pointerCheck["ds_descriptor"] = desc
			pointerCheck["ds_flat_writable_span"] = desc.Base == 0 && desc.Writable && ptr <= desc.Limit && 7 <= desc.Limit-ptr
		}
		if ptr >= 4 && uint64(ptr)+8 <= uint64(len(m.Mem)) {
			tag := binary.LittleEndian.Uint32(m.Mem[ptr-4 : ptr])
			pointerCheck["block_tag"] = tag
			pointerCheck["block_tag_live"] = tag&1 == 1
			pointerCheck["payload_first8_hex"] = fmt.Sprintf("%x", m.Mem[ptr:ptr+8])
			pointerCheck["payload_backed"] = true
			before := m.Mem[ptr]
			pointerCheck["idempotent_payload_write_ok"] = m.Write8(ptr, before) == nil && m.Mem[ptr] == before
		}
		for _, event := range events {
			if event["instruction_eip"] == uint32(0x36d26) {
				before := event["before"].(map[string]any)
				regs := before["registers"].([8]uint32)
				seg := before["segments"].([6]uint16)
				pointerCheck["cdecl_esp_restored"] = m.CPU.R[cpu386.ESP] == regs[cpu386.ESP]+4
				pointerCheck["saved_registers_restored"] = m.CPU.R[cpu386.EBX] == regs[cpu386.EBX] && m.CPU.R[cpu386.ESI] == regs[cpu386.ESI] && m.CPU.R[cpu386.EBP] == regs[cpu386.EBP]
				pointerCheck["saved_segments_restored"] = m.CPU.Seg[cpu386.SegGS] == seg[cpu386.SegGS] && m.CPU.Seg[cpu386.SegFS] == seg[cpu386.SegFS] && m.CPU.Seg[cpu386.SegES] == seg[cpu386.SegES]
			}
		}
	}
	return map[string]any{"classification": "natural LE entry diagnostic; first allocation only; not chapter oracle",
		"returned_state": returnedState, "first_pointer_check": pointerCheck,
		"initial_segments_after_load": initialSegments, "entry_state_after_runtime_install": entryState, "segment_and_saved_gs_events": events,
		"status": status, "steps": steps, "eip": m.CPU.EIP, "eax": m.CPU.R[cpu386.EAX], "esp": m.CPU.R[cpu386.ESP],
		"error": stopped, "stopped_eip": stoppedEIP, "stopped_next8_hex": stoppedBytes,
		"tail_eip": tail, "omitted_hooks": []string{"_nmalloc", "_nfree", "__Init_Argv"}, "fixture_heap": false}
}

// 自然LE入口的有界生命週期診斷；沒有人工heap、輸入或章狀態注入。
func naturalAllocationLifecycle(data []byte, root string) map[string]any {
	m, err := machine.LoadLE(data)
	must(err)
	must(machine.InstallDOS4GWBIOSData(m))
	files, err := machine.OpenDirectoryReadOnlyFiles(root)
	must(err)
	defer files.Close()
	services := machine.NewFD2StartupDOS(files)
	defer services.Close()
	m.CPU.IntHook = services.Handle
	opl := machine.NewLEOPLPorts()
	services.DPMI.RealModeIO = opl
	m.CPU.PortIn, m.CPU.PortOut = opl.In8, opl.Out8
	if !machine.InstallLEVideo(m, opl) {
		panic("LEVideo install failed")
	}
	_, err = machine.InstallFD2WatcomRuntimeWithHeapCapacity(m, 32*1024*1024, services.DPMI)
	must(err)
	if !machine.InstallLEBIOSClock(m, opl) {
		panic("BIOS clock install failed")
	}
	if !machine.InstallLEBIOSKeyboard(m) {
		panic("BIOS keyboard install failed")
	}
	installed := m.CPU.StepHook
	m.CPU.StepHook = func(c *cpu386.CPU) (bool, error) {
		if c.EIP == 0x36d26 || c.EIP == 0x37426 || c.EIP == 0x46114 {
			return false, nil
		}
		return installed(c)
	}
	type frame struct {
		entry, caller, esp, param uint32
		started                   int
	}
	var pending []frame
	var episodes []map[string]any
	var reuseEvents []map[string]any
	allocations, frees, reuses := 0, 0, 0
	allocatedBefore := map[uint32]bool{}
	freedAfterAllocation := map[uint32]int{}
	var tail []uint32
	var stopped string
	var stoppedEIP uint32
	var stoppedBytes string
	steps := 0
	for steps < 1000000 && !services.Exited {
		c := m.CPU
		if len(pending) > 0 {
			f := pending[len(pending)-1]
			if c.EIP == f.caller && c.R[cpu386.ESP] == f.esp+4 {
				kind := "allocation"
				ptr := c.R[cpu386.EAX]
				if f.entry == 0x37426 {
					kind = "free"
					ptr = f.param
				}
				e := map[string]any{"kind": kind, "entry": f.entry, "caller": f.caller, "entry_esp": f.esp, "return_esp": c.R[cpu386.ESP], "parameter": f.param, "eax": c.R[cpu386.EAX], "steps_entry": f.started, "steps_return": steps, "status": "returned"}
				if ptr >= 4 && uint64(ptr)+8 <= uint64(len(m.Mem)) {
					e["pointer"] = ptr
					e["block_tag"] = binary.LittleEndian.Uint32(m.Mem[ptr-4 : ptr])
					e["payload_first8_hex"] = fmt.Sprintf("%x", m.Mem[ptr:ptr+8])
				}
				if kind == "allocation" {
					allocations++
					if ptr != 0 {
						if freeStep, ok := freedAfterAllocation[ptr]; ok {
							reuses++
							if len(reuseEvents) < 64 {
								reuseEvents = append(reuseEvents, map[string]any{"pointer": ptr, "free_return_step": freeStep, "allocation_return_step": steps, "request": f.param, "allocation_caller": f.caller, "block_tag": e["block_tag"]})
							}
							delete(freedAfterAllocation, ptr)
						}
						allocatedBefore[ptr] = true
					}
				} else {
					frees++
					if allocatedBefore[ptr] {
						freedAfterAllocation[ptr] = steps
					}
				}
				if len(episodes) < 256 {
					episodes = append(episodes, e)
				}
				pending = pending[:len(pending)-1]
			}
		}
		if c.EIP == 0x36d26 || c.EIP == 0x37426 {
			desc, ok := c.Descriptors[c.Seg[cpu386.SegSS]]
			if !ok || c.R[cpu386.ESP] > desc.Limit || 7 > desc.Limit-c.R[cpu386.ESP] {
				panic("native lifecycle stack unavailable")
			}
			at := uint64(desc.Base) + uint64(c.R[cpu386.ESP])
			if at+8 > uint64(len(m.Mem)) {
				panic("native lifecycle stack backing unavailable")
			}
			pending = append(pending, frame{c.EIP, binary.LittleEndian.Uint32(m.Mem[at : at+4]), c.R[cpu386.ESP], binary.LittleEndian.Uint32(m.Mem[at+4 : at+8]), steps})
		}
		tail = append(tail, c.EIP)
		if len(tail) > 32 {
			tail = tail[len(tail)-32:]
		}
		before := c.EIP
		if err := c.Step(); err != nil {
			stopped, stoppedEIP = err.Error(), before
			if uint64(before)+8 <= uint64(len(m.Mem)) {
				stoppedBytes = fmt.Sprintf("%x", m.Mem[before:before+8])
			}
			break
		}
		steps++
	}
	var unfinished []map[string]any
	for _, f := range pending {
		unfinished = append(unfinished, map[string]any{"entry": f.entry, "caller": f.caller, "entry_esp": f.esp, "parameter": f.param, "steps_entry": f.started, "status": "pending_at_stop"})
	}
	status := "bounded_limit"
	if stopped != "" {
		status = "stopped"
	}
	if services.Exited {
		status = "program_exited"
	}
	return map[string]any{"platform_profile": "same exported BIOS data / LEVideo / LEOPLPorts / BIOS clock / keyboard setup as versioned apps/fd2/cmd/oracle; existing hardware-spec approximations",
		"file_profile": "OpenDirectoryReadOnlyFiles; pristine root; no writable overlay", "program_exited": services.Exited, "exit_code": services.ExitCode, "console_hex": fmt.Sprintf("%x", services.Console),
		"classification": "natural LE entry allocator lifecycle diagnostic; not chapter oracle", "fixture_heap": false, "omitted_hooks": []string{"_nmalloc", "_nfree", "__Init_Argv"}, "steps": steps, "step_limit": 1000000, "episode_sample_limit": 256, "allocation_returns": allocations, "free_returns": frees, "exact_pointer_reuse_count": reuses, "exact_pointer_reuse_samples": reuseEvents, "status": status, "eip": m.CPU.EIP, "registers": m.CPU.R, "segments": m.CPU.Seg, "episodes": episodes, "unfinished": unfinished, "tail_eip": tail, "error": stopped, "stopped_eip": stoppedEIP, "stopped_next8_hex": stoppedBytes}
}

func main() {
	exe := flag.String("exe", "", "唯讀固定版本原版檔")
	ida := flag.String("ida", "", "同版本IDA原始指令JSON")
	sourceRoot := flag.String("dosgolem-root", "", "唯讀dosgolem來源")
	commit := flag.String("dosgolem-commit", "", "由主機git核對的來源提交")
	gameRoot := flag.String("root", "", "生命週期診斷的原版唯讀資料目錄")
	assetManifest := flag.String("asset-manifest", "", "固定原版資產清單")
	lifecycle := flag.Bool("natural-lifecycle", false, "有界自然配置／釋放鏈診斷，不是章oracle")
	dirty := flag.Bool("dosgolem-dirty", false, "來源工作樹是否有未提交變更")
	out := flag.String("out", "", "新JSON輸出，不覆寫")
	flag.Parse()
	if os.Getenv("FD2_HEAP_PROBE_DOCKER") != "1" || len(*commit) != 40 || *out == "" {
		panic("explicit Docker invocation, commit and output required")
	}
	if _, err := os.Stat("/.dockerenv"); err != nil {
		panic("Docker required")
	}
	data, err := os.ReadFile(*exe)
	must(err)
	if len(data) != 357074 || digest(data) != fixedEXE {
		panic("fixed EXE required")
	}
	idaBytes, err := os.ReadFile(*ida)
	must(err)
	var probe struct {
		SHA256       string `json:"sha256"`
		Tool         string `json:"tool"`
		AddressSpace string `json:"address_space"`
		Functions    []struct {
			Instructions []struct{ Address, Bytes string }
		}
	}
	must(json.Unmarshal(idaBytes, &probe))
	if probe.SHA256 != fixedEXE || probe.Tool != "IDA Pro 9.4" || probe.AddressSpace != "IDA LE linear" || len(probe.Functions) != 3 {
		panic("IDA provenance invalid")
	}
	loaded, err := machine.LoadLE(data)
	must(err)
	matched := 0
	for _, f := range probe.Functions {
		for _, ins := range f.Instructions {
			var at uint32
			_, err := fmt.Sscanf(ins.Address, "0x%x", &at)
			must(err)
			var raw []byte
			for i := 0; i < len(ins.Bytes); i += 2 {
				var value byte
				_, err := fmt.Sscanf(ins.Bytes[i:i+2], "%x", &value)
				must(err)
				raw = append(raw, value)
			}
			if uint64(at)+uint64(len(raw)) > uint64(len(loaded.Mem)) || !bytes.Equal(loaded.Mem[at:at+uint32(len(raw))], raw) {
				panic(fmt.Sprintf("IDA/LE bytes differ at %X", at))
			}
			matched++
		}
	}
	if matched < 200 {
		panic("IDA instruction coverage incomplete")
	}
	var cases []map[string]any
	for _, c := range []struct {
		name         string
		size, prefix uint32
	}{{"rover", 24, 0}, {"prefix_reset", 24, 128}, {"minimum_block", 1, 0}, {"whole_block", 124, 0}, {"no_fit", 129, 0}, {"zero", 0, 0}} {
		cases = append(cases, originalAllocation(data, c.name, c.size, c.prefix))
	}
	sources := map[string]string{}
	for _, name := range []string{"internal/cpu386/cpu.go", "internal/machine/watcom_runtime.go", "internal/machine/le_machine.go", "internal/machine/le_startup.go", "apps/fd2/cmd/oracle/main.go"} {
		b, err := os.ReadFile(filepath.Join(*sourceRoot, name))
		must(err)
		sources[name] = digest(b)
	}
	report := map[string]any{"schema_version": 1, "kind": "fd2_original_allocator_local_probe", "exe_sha256": fixedEXE,
		"go_version": runtime.Version(), "dosgolem_commit_supplied_by_host": *commit, "dosgolem_dirty_supplied_by_host": *dirty, "source_sha256": sources,
		"ida_sha256": digest(idaBytes), "ida_instruction_bytes_matched": matched, "allocator_cases": cases,
		"current_hook": currentHook(data), "natural_first_allocation": naturalFirstAllocation(data),
		"limitations": "局部人工free list與初次配置診斷；未重跑ch18原計畫、未改production、未取得實機或章PLAYER-E2。"}
	if *lifecycle {
		if *gameRoot == "" || *assetManifest == "" {
			panic("lifecycle requires original root and fixed asset manifest")
		}
		manifestBytes, err := os.ReadFile(*assetManifest)
		must(err)
		var manifest struct {
			Files []struct {
				File        string
				Size        int
				MD5, SHA256 string
			}
		}
		must(json.Unmarshal(manifestBytes, &manifest))
		if len(manifest.Files) == 0 {
			panic("empty asset manifest")
		}
		for _, f := range manifest.Files {
			if filepath.Base(f.File) != f.File || f.File == "FD2.SAV" || f.File == "FD2.TMP" {
				panic("invalid fixed asset name")
			}
			raw, err := os.ReadFile(filepath.Join(*gameRoot, f.File))
			must(err)
			if len(raw) != f.Size || digest(raw) != f.SHA256 || fmt.Sprintf("%x", md5.Sum(raw)) != f.MD5 {
				panic("fixed asset mismatch: " + f.File)
			}
		}
		report["asset_manifest"] = map[string]any{"path": *assetManifest, "sha256": digest(manifestBytes), "fixed_files_matched": len(manifest.Files), "files": manifest.Files}
		report["natural_allocation_lifecycle"] = naturalAllocationLifecycle(data, *gameRoot)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	must(err)
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	must(err)
	_, err = f.Write(append(raw, '\n'))
	must(err)
	must(f.Close())
	fmt.Println("local allocator probe written", *out)
}
