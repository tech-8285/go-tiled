/*
Copyright (c) 2026 Lauris Bukšis <lauris@nix.lv>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package render

import (
	"image/color"
	"testing"

	"github.com/lafriks/go-tiled"
)

func renderMapFile(t *testing.T, path string) *Renderer {
	t.Helper()

	tiledMap, err := tiled.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	renderer, err := NewRenderer(tiledMap)
	if err != nil {
		t.Fatal(err)
	}

	if err = renderer.RenderMap(); err != nil {
		t.Fatal(err)
	}

	return renderer
}

func assertPixel(t *testing.T, r *Renderer, x, y int, want color.NRGBA, delta uint8) {
	t.Helper()

	got := color.NRGBAModel.Convert(r.Result.At(x, y)).(color.NRGBA)
	close := func(a, b uint8) bool {
		if a > b {
			return a-b <= delta
		}
		return b-a <= delta
	}
	if !close(got.R, want.R) || !close(got.G, want.G) || !close(got.B, want.B) || !close(got.A, want.A) {
		t.Errorf("pixel (%d,%d) = %v, want %v (±%d)", x, y, got, want, delta)
	}
}

// TestRenderer_RenderMap exercises whole-map compositing on a fixture that
// interleaves a tile layer, an image layer, a nested group (with offset,
// opacity and tint) and an object group in document order.
func TestRenderer_RenderMap(t *testing.T) {
	renderer := renderMapFile(t, "../assets/test_render_map.tmx")

	if got := renderer.Result.Bounds().Dx(); got != 32 {
		t.Errorf("image width = %d, want 32", got)
	}
	if got := renderer.Result.Bounds().Dy(); got != 32 {
		t.Errorf("image height = %d, want 32", got)
	}

	red := color.NRGBA{255, 0, 0, 255}

	// Ground layer only.
	assertPixel(t, renderer, 2, 2, red, 0)
	assertPixel(t, renderer, 20, 20, red, 0)
	// The hidden image layer inside the group must not be drawn.
	assertPixel(t, renderer, 10, 4, red, 0)

	// Group tile (white tinted to green, group opacity 0.5, nested offsets
	// 8+8) over the red ground, where the image layer is transparent.
	assertPixel(t, renderer, 10, 10, color.NRGBA{128, 127, 0, 255}, 2)
	// Group tile over the half-transparent green image layer.
	assertPixel(t, renderer, 13, 10, color.NRGBA{64, 191, 0, 255}, 2)
	// Transparent corner of the group tile lets the image layer show through.
	assertPixel(t, renderer, 14, 14, color.NRGBA{128, 127, 0, 255}, 2)
	// Image layer's own transparent corner leaves the red ground untouched.
	assertPixel(t, renderer, 9, 9, color.NRGBA{128, 127, 0, 255}, 2)

	// Top tile layer (white tile) drawn above the object group's red tile.
	assertPixel(t, renderer, 26, 26, color.NRGBA{255, 255, 255, 255}, 0)
	// The white tile's transparent corner reveals the object tile below it.
	assertPixel(t, renderer, 30, 30, red, 0)
}

// TestRenderer_RenderMap_Orientations makes sure whole-map rendering works
// for all four supported map orientations.
func TestRenderer_RenderMap_Orientations(t *testing.T) {
	tests := []struct {
		file       string
		wantWidth  int
		wantHeight int
	}{
		{"../assets/test_wangsets_map.tmx", 1600, 1600},
		{"../assets/test_isometric.tmx", 800, 400},
		{"../assets/hex.tmx", 336, 248},
		{"../assets/staggered.tmx", 336, 176},
	}

	for _, tt := range tests {
		renderer := renderMapFile(t, tt.file)
		if got := renderer.Result.Bounds().Dx(); got != tt.wantWidth {
			t.Errorf("%s: image width = %d, want %d", tt.file, got, tt.wantWidth)
		}
		if got := renderer.Result.Bounds().Dy(); got != tt.wantHeight {
			t.Errorf("%s: image height = %d, want %d", tt.file, got, tt.wantHeight)
		}
	}
}

// TestRenderer_RenderMap_LegacyCompatible verifies that for maps without
// groups, image layers, offsets or tint colors, RenderMap produces exactly
// the same image as the legacy per-layer rendering.
func TestRenderer_RenderMap_LegacyCompatible(t *testing.T) {
	tiledMap, err := tiled.LoadFile("../assets/test_wangsets_map.tmx")
	if err != nil {
		t.Fatal(err)
	}

	legacy, err := NewRenderer(tiledMap)
	if err != nil {
		t.Fatal(err)
	}
	if err = legacy.RenderVisibleLayers(); err != nil {
		t.Fatal(err)
	}

	whole, err := NewRenderer(tiledMap)
	if err != nil {
		t.Fatal(err)
	}
	if err = whole.RenderMap(); err != nil {
		t.Fatal(err)
	}

	bounds := legacy.Result.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if got, want := whole.Result.NRGBAAt(x, y), legacy.Result.NRGBAAt(x, y); got != want {
				t.Fatalf("pixel (%d,%d) = %v, want %v", x, y, got, want)
			}
		}
	}
}
