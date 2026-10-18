// Copyright 2014 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Adapted from github.com/hajimehoshi/ebiten/v2 v2.6.6 gameforui.go:
// screenShaderSrc and gameForUI.DrawFinalScreen. Changes: use the supplied
// FinalScreenDrawer geometry and lazily compile the shader. The shader and
// filter selection retain that version's default behavior.
// Release packaging preserves Ebiten's full LICENSE under third-party/.

package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const defaultScreenShaderSource = `//kage:unit pixels

package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	// Blend source colors in a square region, which size is 1/scale.
	scale := imageDstSize()/imageSrc0Size()
	pos := texCoord
	p0 := pos - 1/2.0/scale
	p1 := pos + 1/2.0/scale

	// Texels must be in the source rect, so it is not necessary to check.
	c0 := imageSrc0UnsafeAt(p0)
	c1 := imageSrc0UnsafeAt(vec2(p1.x, p0.y))
	c2 := imageSrc0UnsafeAt(vec2(p0.x, p1.y))
	c3 := imageSrc0UnsafeAt(p1)

	// p is the p1 value in one pixel assuming that the pixel's upper-left is (0, 0) and the lower-right is (1, 1).
	rate := clamp(fract(p1)*scale, 0, 1)
	return mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y)
}
`

func (p *modernHDPresentation) drawDefaultFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	scale := geometry.Element(0, 0)
	switch {
	case !ebiten.IsScreenFilterEnabled(), math.Floor(scale) == scale:
		screen.DrawImage(offscreen, &ebiten.DrawImageOptions{GeoM: geometry})
	case scale < 1:
		screen.DrawImage(offscreen, &ebiten.DrawImageOptions{GeoM: geometry, Filter: ebiten.FilterLinear})
	default:
		if p.screenShader == nil {
			shader, err := ebiten.NewShader([]byte(defaultScreenShaderSource))
			if err != nil {
				panic(err)
			}
			p.screenShader = shader
		}
		op := &ebiten.DrawRectShaderOptions{GeoM: geometry}
		op.Images[0] = offscreen
		screen.DrawRectShader(offscreen.Bounds().Dx(), offscreen.Bounds().Dy(), p.screenShader, op)
	}
}
