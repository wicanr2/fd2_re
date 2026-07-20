// Package afm 執行期解碼 ANI.DAT 的 AFM 動畫(開場過場)。
//
// AFM(Animation File Manager v1.00,Lo Yuan Tsung 1993)不是逐幀點陣圖,而是
// 10-opcode 的增量繪圖 VM(反組譯 doc39):每幀是一段 script,對「上一幀遺留的
// framebuffer/palette」疊加(填色/RLE/局部貼圖/局部調色盤),不清空重畫。
// 本套件把 tools/decode_ani.py 的 VM 忠實移植成 Go,直接解玩家自備的 ANI.DAT,
// 不夾帶任何預解版權幀(素材本機保留紀律)。
//
// 反組譯位址(FD2.EXE linear):播放器 0x020421、VM 派發 0x36c9e、跳表 0x5276a、
// framebuffer=VGA 0xA0000(無雙緩衝)。VGA 6-bit palette 以 (v<<2)|(v>>4) 還原 8-bit。
package afm

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"os"
)

const (
	scrW       = 320
	scrH       = 200
	frameBytes = scrW * scrH // 64000
)

// Clip 是一個 AFM 資源解出的完整影格序列(已套 palette 的 RGBA)。
type Clip struct {
	Title         string
	Frames        []*image.RGBA
	IndexedFrames [][]byte // 320x200 palette indices, one immutable snapshot/frame
	Palettes      [][]byte // matching 768-byte VGA 6-bit palette snapshots
}

// LoadANI 開啟一個 .DAT 容器(LLLLLL magic + uint32 offset 目錄),回傳其資源數。
func containerEntries(data []byte) ([][]byte, error) {
	if len(data) < 10 || string(data[:6]) != "LLLLLL" {
		return nil, errors.New("afm: 缺少 LLLLLL magic")
	}
	first := binary.LittleEndian.Uint32(data[6:])
	if first < 6 || int(first) > len(data) || (first-6)%4 != 0 {
		return nil, errors.New("afm: 目錄起點不合理")
	}
	n := int((first - 6) / 4)
	offs := make([]int, n+1)
	for i := 0; i < n; i++ {
		offs[i] = int(binary.LittleEndian.Uint32(data[6+4*i:]))
	}
	offs[n] = len(data)
	out := make([][]byte, n)
	for i := 0; i < n; i++ {
		if offs[i] > offs[i+1] || offs[i+1] > len(data) {
			return nil, errors.New("afm: 目錄越界")
		}
		out[i] = data[offs[i]:offs[i+1]]
	}
	return out, nil
}

// DecodeResource 解一個 AFM 資源為 Clip。res 是容器內的資源 index。
func DecodeResource(datPath string, res int) (*Clip, error) {
	raw, err := os.ReadFile(datPath)
	if err != nil {
		return nil, err
	}
	entries, err := containerEntries(raw)
	if err != nil {
		return nil, err
	}
	if res < 0 || res >= len(entries) {
		return nil, errors.New("afm: 資源 index 越界")
	}
	return decodeAFM(entries[res])
}

// decodeAFM 解單一 AFM 檔(資源)。標頭 +0xA5=frameCount,+0xAD 起 N×(8B 幀記錄+script)。
func decodeAFM(d []byte) (*Clip, error) {
	if len(d) < 0xAD {
		return nil, errors.New("afm: 資源過短")
	}
	title := ""
	if len(d) >= 0xA1 {
		title = string(d[0x51:0xA1])
	}
	frameCount := int(binary.LittleEndian.Uint16(d[0xA5:]))
	clip := &Clip{Title: title}
	pal := make([]byte, 768)
	fb := make([]byte, frameBytes)
	pos := 0xAD
	for i := 0; i < frameCount; i++ {
		if pos+8 > len(d) {
			break
		}
		compSize := int(binary.LittleEndian.Uint16(d[pos:]))
		cmdCount := int(binary.LittleEndian.Uint16(d[pos+2:]))
		pos += 8
		if pos+compSize > len(d) {
			break
		}
		script := d[pos : pos+compSize]
		pos += compSize
		if err := runVM(script, cmdCount, pal, fb); err != nil {
			break // 解碼中斷:保留前面已成功的幀(同 python 解碼器行為)
		}
		clip.IndexedFrames = append(clip.IndexedFrames, append([]byte(nil), fb...))
		clip.Palettes = append(clip.Palettes, append([]byte(nil), pal...))
		clip.Frames = append(clip.Frames, toRGBA(fb, pal))
	}
	if len(clip.Frames) == 0 {
		return nil, errors.New("afm: 零幀")
	}
	return clip, nil
}

// toRGBA 把 index framebuffer + 6-bit palette 轉成 RGBA。
func toRGBA(fb, pal []byte) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, scrW, scrH))
	for i, ci := range fb {
		r := pal[int(ci)*3]
		g := pal[int(ci)*3+1]
		b := pal[int(ci)*3+2]
		img.Pix[i*4] = (r << 2) | (r >> 4)
		img.Pix[i*4+1] = (g << 2) | (g >> 4)
		img.Pix[i*4+2] = (b << 2) | (b >> 4)
		img.Pix[i*4+3] = 0xff
	}
	return img
}

var _ = color.RGBA{} // color 保留供未來擴充

// rle2mode 是 opcode 2/6 共用的 2-mode RLE:高 2bit==0b11 → run(len=ctrl&0x3F),否則單一 literal。
func rle2mode(data []byte, pos int, dst []byte, count int) (int, error) {
	written := 0
	for written < count {
		if pos >= len(data) {
			return pos, errors.New("afm: RLE 越界")
		}
		ctrl := data[pos]
		pos++
		if ctrl&0xC0 == 0xC0 {
			n := int(ctrl & 0x3F)
			if pos >= len(data) {
				return pos, errors.New("afm: RLE run 值越界")
			}
			val := data[pos]
			pos++
			for i := 0; i < n && written < count; i++ {
				dst[written] = val
				written++
			}
		} else {
			dst[written] = ctrl
			written++
		}
	}
	return pos, nil
}

// runVM 執行 cmdCount 個 AFM 指令,原地改 pal(768B)與 fb(64000B)。忠實 doc39 opcode 表。
func runVM(s []byte, cmdCount int, pal, fb []byte) error {
	pos := 0
	rd8 := func() (byte, error) {
		if pos >= len(s) {
			return 0, errors.New("afm: script 越界")
		}
		v := s[pos]
		pos++
		return v, nil
	}
	rd16 := func() (int, error) {
		if pos+2 > len(s) {
			return 0, errors.New("afm: script 越界(u16)")
		}
		v := int(binary.LittleEndian.Uint16(s[pos:]))
		pos += 2
		return v, nil
	}
	for c := 0; c < cmdCount; c++ {
		op, err := rd8()
		if err != nil {
			return err
		}
		switch op {
		case 0: // palette 全填
			v, e := rd8()
			if e != nil {
				return e
			}
			for i := range pal {
				pal[i] = v
			}
		case 1: // palette 字面載入
			if pos+768 > len(s) {
				return errors.New("afm: op1 越界")
			}
			copy(pal, s[pos:pos+768])
			pos += 768
		case 2: // palette RLE
			pos, err = rle2mode(s, pos, pal, 768)
			if err != nil {
				return err
			}
		case 3: // palette 局部貼補:N×(idx,cnt,RGB×cnt)
			n, e := rd8()
			if e != nil {
				return e
			}
			for k := 0; k < int(n); k++ {
				idx, e1 := rd8()
				cnt, e2 := rd8()
				if e1 != nil || e2 != nil {
					return errors.New("afm: op3 越界")
				}
				off, length := int(idx)*3, int(cnt)*3
				if pos+length > len(s) || off+length > 768 {
					return errors.New("afm: op3 資料越界")
				}
				copy(pal[off:off+length], s[pos:pos+length])
				pos += length
			}
		case 4: // framebuffer 全填
			v, e := rd8()
			if e != nil {
				return e
			}
			for i := range fb {
				fb[i] = v
			}
		case 5: // framebuffer 字面載入
			if pos+frameBytes > len(s) {
				return errors.New("afm: op5 越界")
			}
			copy(fb, s[pos:pos+frameBytes])
			pos += frameBytes
		case 6: // framebuffer RLE(主力全螢幕)
			pos, err = rle2mode(s, pos, fb, frameBytes)
			if err != nil {
				return err
			}
		case 7: // 單點繪製 ×N
			n, e := rd16()
			if e != nil {
				return e
			}
			for k := 0; k < n; k++ {
				off, e1 := rd16()
				val, e2 := rd8()
				if e1 != nil || e2 != nil {
					return errors.New("afm: op7 越界")
				}
				if off >= 0 && off < frameBytes {
					fb[off] = val
				}
			}
		case 8: // 區段填色 ×N
			n, e := rd16()
			if e != nil {
				return e
			}
			for k := 0; k < n; k++ {
				off, e1 := rd16()
				length, e2 := rd8()
				val, e3 := rd8()
				if e1 != nil || e2 != nil || e3 != nil {
					return errors.New("afm: op8 越界")
				}
				end := off + int(length)
				if end > frameBytes {
					end = frameBytes
				}
				for i := off; i < end; i++ {
					fb[i] = val
				}
			}
		case 9: // 區段貼圖 ×N
			n, e := rd16()
			if e != nil {
				return e
			}
			for k := 0; k < n; k++ {
				off, e1 := rd16()
				length, e2 := rd8()
				if e1 != nil || e2 != nil {
					return errors.New("afm: op9 越界")
				}
				end := off + int(length)
				if end > frameBytes {
					end = frameBytes
				}
				if pos+int(length) > len(s) {
					return errors.New("afm: op9 資料越界")
				}
				copy(fb[off:end], s[pos:pos+(end-off)])
				pos += int(length)
			}
		default:
			return errors.New("afm: 未知 opcode")
		}
	}
	return nil
}
