//go:build ignore

// 在 Docker 的臨時 module 中以 go run main.go 執行。這是 #102 的原始
// 配置器局部診斷，不是章 oracle；人工 free list 與自然啟動分開記錄。
package main

import (
	"bytes"
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
	steps := 0
	var tail []uint32
	var stopped string
	for steps < 100000 && m.CPU.EIP != 0x4cc51 {
		tail = append(tail, m.CPU.EIP)
		if len(tail) > 32 {
			tail = tail[len(tail)-32:]
		}
		if err := m.CPU.Step(); err != nil {
			stopped = err.Error()
			break
		}
		steps++
	}
	status := "stopped"
	if m.CPU.EIP == 0x4cc51 {
		status = "first_caller_returned"
	}
	return map[string]any{"classification": "natural LE entry diagnostic; first allocation only; not chapter oracle",
		"status": status, "steps": steps, "eip": m.CPU.EIP, "eax": m.CPU.R[cpu386.EAX], "esp": m.CPU.R[cpu386.ESP],
		"error": stopped, "tail_eip": tail, "omitted_hooks": []string{"_nmalloc", "_nfree", "__Init_Argv"}, "fixture_heap": false}
}

func main() {
	exe := flag.String("exe", "", "唯讀固定版本原版檔")
	ida := flag.String("ida", "", "同版本IDA原始指令JSON")
	sourceRoot := flag.String("dosgolem-root", "", "唯讀dosgolem來源")
	commit := flag.String("dosgolem-commit", "", "由主機git核對的來源提交")
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
	for _, name := range []string{"internal/cpu386/cpu.go", "internal/machine/watcom_runtime.go", "internal/machine/le_machine.go", "apps/fd2/cmd/oracle/main.go"} {
		b, err := os.ReadFile(filepath.Join(*sourceRoot, name))
		must(err)
		sources[name] = digest(b)
	}
	report := map[string]any{"schema_version": 1, "kind": "fd2_original_allocator_local_probe", "exe_sha256": fixedEXE,
		"go_version": runtime.Version(), "dosgolem_commit_supplied_by_host": *commit, "dosgolem_dirty_supplied_by_host": *dirty, "source_sha256": sources,
		"ida_sha256": digest(idaBytes), "ida_instruction_bytes_matched": matched, "allocator_cases": cases,
		"current_hook": currentHook(data), "natural_first_allocation": naturalFirstAllocation(data),
		"limitations": "局部人工free list與初次配置診斷；未重跑ch18原計畫、未改production、未取得實機或章PLAYER-E2。"}
	raw, err := json.MarshalIndent(report, "", "  ")
	must(err)
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	must(err)
	_, err = f.Write(append(raw, '\n'))
	must(err)
	must(f.Close())
	fmt.Println("local allocator probe written", *out)
}
