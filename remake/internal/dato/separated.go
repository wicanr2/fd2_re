package dato

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
)

const mouthFrameCount = 4

// LoadSeparatedResource 讀取離線匯出的四張 indexed PNG。它只接受
// *image.Paletted，避免由 RGBA 顏色猜回原始 VGA index。
func LoadSeparatedResource(portraitRoot string, resource int) ([]Frame, error) {
	if portraitRoot == "" || resource < 0 {
		return nil, fmt.Errorf("dato: separated portrait input is invalid")
	}
	frames := make([]Frame, mouthFrameCount)
	for mouth := 0; mouth < mouthFrameCount; mouth++ {
		path := filepath.Join(portraitRoot, fmt.Sprintf("DATO_%03d_m%d.png", resource, mouth))
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("dato: separated portrait %d frame %d: %w", resource, mouth, err)
		}
		decoded, _, decodeErr := image.Decode(file)
		closeErr := file.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("dato: separated portrait %d frame %d decode: %w", resource, mouth, decodeErr)
		}
		if closeErr != nil {
			return nil, closeErr
		}
		paletted, ok := decoded.(*image.Paletted)
		if !ok {
			return nil, fmt.Errorf("dato: separated portrait %d frame %d is not indexed PNG", resource, mouth)
		}
		width, height := paletted.Bounds().Dx(), paletted.Bounds().Dy()
		if width <= 0 || height <= 0 || paletted.Stride < width || len(paletted.Pix) < paletted.Stride*height {
			return nil, fmt.Errorf("dato: separated portrait %d frame %d geometry is invalid", resource, mouth)
		}
		pixels := make([]byte, width*height)
		for row := 0; row < height; row++ {
			copy(pixels[row*width:(row+1)*width], paletted.Pix[row*paletted.Stride:row*paletted.Stride+width])
		}
		frames[mouth] = Frame{Width: width, Height: height, Pixels: pixels}
	}
	return frames, nil
}
