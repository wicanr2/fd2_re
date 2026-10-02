package fdother

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	NativeMapParallaxWidth  = 462
	NativeMapParallaxHeight = 226
)

// NativeMapParallaxSurface 是 sub_10652 raw17 分支的 [0x53AFF]。
// 定位、原始 bytes 與分級見 docs/data/ida/fd2_ch18_parallax_backdrop_20261002.json。
type NativeMapParallaxSurface struct {
	Pixels []byte
}

var nativeMapParallaxIdentities = [...]struct {
	resource, size int
	md5, sha256    string
}{
	{16, 1950, "20c9979e017049b845a29219c3627d8c", "fced46effbd2a7c0deffa0a820143d9658ee6c5f5cd8a2103e92fbf0c5245add"},
	{17, 34664, "829b6a2831831b80d3026dbce10a5a99", "10b3e4ca5c8e43f76fdbeb3adfa7b493eec4e8ca1f8b44b03d15092d811e979f"},
}

// LoadSeparatedNativeMapParallax 嚴格載入原版 #16/#17 的分離圖面。
// 其他 raw chapter 的雙段資源尚未由此契約開放。
func LoadSeparatedNativeMapParallax(root string) (*NativeMapParallaxSurface, error) {
	frames := make([]Frame, len(nativeMapParallaxIdentities))
	for i, identity := range nativeMapParallaxIdentities {
		path := filepath.Join(root, fmt.Sprintf("FDOTHER_%03d", identity.resource), "resource.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var doc separatedSurfaceDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, err
		}
		if doc.Source.RawSize != identity.size || doc.Source.RawMD5 != identity.md5 || doc.Source.RawSHA256 != identity.sha256 {
			return nil, errors.New("fdother: map parallax raw identity mismatch")
		}
		frame, err := LoadSeparatedSingleFrame(root, "FDOTHER.DAT", identity.resource)
		if err != nil {
			return nil, err
		}
		frames[i] = frame
	}
	return BuildNativeMapParallax(frames[0], frames[1])
}

// BuildNativeMapParallax 重現 0x10773／0x107AF 的上下兩段解碼。
// 固定資源兩段均完整不透明，禁止用零值補入未定義的 malloc 像素。
func BuildNativeMapParallax(top, bottom Frame) (*NativeMapParallaxSurface, error) {
	surface := make([]byte, NativeMapParallaxWidth*NativeMapParallaxHeight)
	for i, frame := range []Frame{top, bottom} {
		if frame.X != 0 || frame.Y != 0 || frame.Width != NativeMapParallaxWidth || frame.Height != NativeMapParallaxHeight/2 {
			return nil, errors.New("fdother: map parallax segment geometry mismatch")
		}
		pixels, mask, err := frame.IndexedLayers()
		if err != nil {
			return nil, err
		}
		for _, opacity := range mask {
			if opacity != 255 {
				return nil, errors.New("fdother: map parallax segment is not fully opaque")
			}
		}
		copy(surface[i*len(pixels):], pixels)
	}
	return &NativeMapParallaxSurface{Pixels: surface}, nil
}

// BlitNativeMapParallaxViewport 是 sub_11EEE 0x12024..0x120FE 的 copy。
// scrollX/Y 保留 [0x53B07]/[0x53B0B] 原始值，整格靜止時為零。
func BlitNativeMapParallaxViewport(dst []byte, stride int, surface *NativeMapParallaxSurface, cameraX, cameraY, scrollX, scrollY int) error {
	if surface == nil || len(surface.Pixels) != NativeMapParallaxWidth*NativeMapParallaxHeight ||
		stride < 312 || len(dst) < 312 || stride > (len(dst)-312)/191 ||
		cameraX < 0 || cameraX > 50 || cameraY < 0 || cameraY > 17 || scrollX < -6 || scrollX > 6 || scrollY < -6 || scrollY > 6 {
		return errors.New("fdother: incomplete map parallax viewport")
	}
	x, y := 3*cameraX+scrollX/2, 2*cameraY+scrollY/3
	if x < 0 || x+312 > NativeMapParallaxWidth || y < 0 || y+192 > NativeMapParallaxHeight {
		return errors.New("fdother: map parallax viewport exceeds source")
	}
	for row := 0; row < 192; row++ {
		src := (y+row)*NativeMapParallaxWidth + x
		copy(dst[row*stride:row*stride+312], surface.Pixels[src:src+312])
	}
	return nil
}
