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
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lafriks/go-tiled"
)

// Distinct, fully opaque colors used to tell frame tiles apart by pixel value.
var (
	colorRed   = color.NRGBA{R: 255, A: 255}
	colorGreen = color.NRGBA{G: 255, A: 255}
	colorBlue  = color.NRGBA{B: 255, A: 255}
	colorCyan  = color.NRGBA{G: 255, B: 255, A: 255}
	colorWhite = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
)

// markPixel paints one white pixel at a known asymmetric spot (top-left), so
// horizontal flips are observable in the rendered output.
func markPixel(img *image.NRGBA) {
	img.SetNRGBA(1, 1, colorWhite)
}

func pngBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// solidImage returns a size-sized NRGBA image filled with c.
func solidImage(size image.Rectangle, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(size)
	for y := size.Min.Y; y < size.Max.Y; y++ {
		for x := size.Min.X; x < size.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// animationTestFS builds an in-memory map with two tilesets:
//   - "atlas": one PNG with four 8x8 tiles (ids 0..3, red/green/blue/cyan);
//     tile 0 animates 0 -> 1 (200ms each), tile 2 (blue) animates 2 -> 3.
//   - "collection": four separate 8x8 images with the same colors and the same
//     animations, so parity between atlas and image-collection rendering can be
//     asserted pixel for pixel.
//
// The map places animated tiles (plain and horizontally flipped) on a tile
// layer and as tile objects (plain and rotated 90°).
func animationTestFS(t *testing.T) fstest.MapFS {
	t.Helper()

	// 4x1 atlas of 8x8 tiles.
	atlas := image.NewNRGBA(image.Rect(0, 0, 32, 8))
	colors := []color.NRGBA{colorRed, colorGreen, colorBlue, colorCyan}
	for i, c := range colors {
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				atlas.SetNRGBA(i*8+x, y, c)
			}
		}
	}
	// Marker on tile 0 (red) at its top-left interior pixel.
	markPixel(atlas)

	collectionImages := make([][]byte, 4)
	for i, c := range colors {
		img := solidImage(image.Rect(0, 0, 8, 8), c)
		if i == 0 {
			markPixel(img)
		}
		collectionImages[i] = pngBytes(t, img)
	}

	atlasTSX := `<?xml version="1.0" encoding="UTF-8"?>
<tileset name="atlas" tilewidth="8" tileheight="8" tilecount="4" columns="4">
 <image source="atlas.png" width="32" height="8"/>
 <tile id="0">
  <animation>
   <frame tileid="0" duration="200"/>
   <frame tileid="1" duration="200"/>
  </animation>
 </tile>
 <tile id="2">
  <animation>
   <frame tileid="2" duration="200"/>
   <frame tileid="3" duration="200"/>
  </animation>
 </tile>
</tileset>`

	collectionTSX := `<?xml version="1.0" encoding="UTF-8"?>
<tileset name="collection" tilewidth="8" tileheight="8" tilecount="4" columns="1">
 <tile id="0"><image source="c0.png" width="8" height="8"/>
  <animation>
   <frame tileid="0" duration="200"/>
   <frame tileid="1" duration="200"/>
  </animation>
 </tile>
 <tile id="1"><image source="c1.png" width="8" height="8"/></tile>
 <tile id="2"><image source="c2.png" width="8" height="8"/>
  <animation>
   <frame tileid="2" duration="200"/>
   <frame tileid="3" duration="200"/>
  </animation>
 </tile>
 <tile id="3"><image source="c3.png" width="8" height="8"/></tile>
</tileset>`

	// GID 1..4 = atlas tiles 0..3; GID 5..8 = collection tiles 0..3.
	// Horizontal flip of GID 1 (atlas tile 0) is 0x80000000|1 = 2147483649.
	//
	// Layout (each cell/object is 8px):
	//   row 0 (y 0..8):   tile layer cells (plain, flipped, two collections)
	//   row 1 (y 8..16):  collection tile object at (0,8)
	//   rows 2..4:        plain object at (0,24), rotated 90deg object at
	//                     (16,24), each anchored bottom-left like Tiled.
	tmx := `<?xml version="1.0" encoding="UTF-8"?>
<map version="1.2" orientation="orthogonal" renderorder="right-down" width="4" height="5" tilewidth="8" tileheight="8" infinite="0" nextlayerid="3" nextobjectid="4">
 <tileset firstgid="1" source="atlas.tsx"/>
 <tileset firstgid="5" source="collection.tsx"/>
 <layer id="1" name="tiles" width="4" height="5">
  <data encoding="csv">
1,2147483649,5,7,
0,0,0,0,
0,0,0,0,
0,0,0,0,
0,0,0,0
</data>
 </layer>
 <objectgroup id="2" name="objects">
  <object id="1" gid="1" x="0" y="24" width="8" height="8"/>
  <object id="2" gid="1" rotation="90" x="16" y="24" width="8" height="8"/>
  <object id="3" gid="5" x="0" y="8" width="8" height="8"/>
 </objectgroup>
</map>`

	fsys := fstest.MapFS{
		"map.tmx":        {Data: []byte(tmx)},
		"atlas.tsx":      {Data: []byte(atlasTSX)},
		"atlas.png":      {Data: pngBytes(t, atlas)},
		"collection.tsx": {Data: []byte(collectionTSX)},
		"c0.png":         {Data: collectionImages[0]},
		"c1.png":         {Data: collectionImages[1]},
		"c2.png":         {Data: collectionImages[2]},
		"c3.png":         {Data: collectionImages[3]},
	}
	return fsys
}

func loadAnimationMap(t *testing.T, fsys fstest.MapFS) (*tiled.Map, *Renderer) {
	t.Helper()
	m, err := tiled.LoadFile("map.tmx", tiled.WithFileSystem(fsys))
	if err != nil {
		t.Fatalf("load map: %v", err)
	}
	r, err := NewRendererWithFileSystem(m, fsys)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	return m, r
}

func TestRenderer_Animation_TileLayerFrames(t *testing.T) {
	fsys := animationTestFS(t)

	tests := []struct {
		name        string
		at          time.Duration
		wantFirst   color.NRGBA // cell (0,0): atlas tile 0 at frames red -> green
		wantFlipped color.NRGBA // cell (1,0): same tile, horizontally flipped
		wantCollect color.NRGBA // cell (2,0): collection tile 0, red -> green
		wantBlue    color.NRGBA // cell (3,0): collection tile 2, blue -> cyan
	}{
		{"first frame", 0, colorRed, colorRed, colorRed, colorBlue},
		{"still first frame at 199ms", 199 * time.Millisecond, colorRed, colorRed, colorRed, colorBlue},
		{"second frame at 200ms", 200 * time.Millisecond, colorGreen, colorGreen, colorGreen, colorCyan},
		{"loop wraps at 400ms", 400 * time.Millisecond, colorRed, colorRed, colorRed, colorBlue},
		{"stable in later loop", 600 * time.Millisecond, colorGreen, colorGreen, colorGreen, colorCyan},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, r := loadAnimationMap(t, fsys)
			r.SetAnimationTime(tt.at)
			if err := r.RenderVisibleLayers(); err != nil {
				t.Fatalf("render layers: %v", err)
			}

			// Center pixels of each 8x8 cell are unaffected by the marker.
			checks := []struct {
				x, y int
				want color.NRGBA
			}{
				{4, 4, tt.wantFirst},
				{12, 4, tt.wantFlipped},
				{20, 4, tt.wantCollect},
				{28, 4, tt.wantBlue},
			}
			for _, c := range checks {
				if got := r.Result.NRGBAAt(c.x, c.y); got != c.want {
					t.Errorf("pixel (%d,%d) = %v, want %v", c.x, c.y, got, c.want)
				}
			}
		})
	}
}

func TestRenderer_Animation_PreservesHorizontalFlip(t *testing.T) {
	fsys := animationTestFS(t)
	_, r := loadAnimationMap(t, fsys)
	r.SetAnimationTime(0)
	if err := r.RenderVisibleLayers(); err != nil {
		t.Fatalf("render: %v", err)
	}

	// Marker lives at local (1,1) of the red tile. In cell (0,0) it stays at
	// global (1,1); in the flipped cell (1,0) it mirrors to local (6,1),
	// global (14,1).
	if got := r.Result.NRGBAAt(1, 1); got != colorWhite {
		t.Errorf("unflipped marker = %v, want white", got)
	}
	if got := r.Result.NRGBAAt(14, 1); got != colorWhite {
		t.Errorf("flipped marker = %v, want white (marker must mirror horizontally)", got)
	}
	// The mirrored spot in the unflipped cell must just be red.
	if got := r.Result.NRGBAAt(6, 1); got != colorRed {
		t.Errorf("unflipped mirrored spot = %v, want red", got)
	}
}

func TestRenderer_Animation_ObjectGroups(t *testing.T) {
	fsys := animationTestFS(t)

	t.Run("tile object swaps frames", func(t *testing.T) {
		_, r := loadAnimationMap(t, fsys)
		r.SetAnimationTime(0)
		if err := r.RenderVisibleObjectGroups(); err != nil {
			t.Fatalf("render objects: %v", err)
		}
		// Object 1 occupies (0,16)-(8,24); anchor is bottom-left.
		if got := r.Result.NRGBAAt(4, 20); got != colorRed {
			t.Errorf("object at t=0 = %v, want red", got)
		}

		_, r = loadAnimationMap(t, fsys)
		r.SetAnimationTime(200 * time.Millisecond)
		if err := r.RenderVisibleObjectGroups(); err != nil {
			t.Fatalf("render objects: %v", err)
		}
		if got := r.Result.NRGBAAt(4, 20); got != colorGreen {
			t.Errorf("object at t=200ms = %v, want green", got)
		}
	})

	t.Run("collection tile object matches atlas", func(t *testing.T) {
		_, r := loadAnimationMap(t, fsys)
		r.SetAnimationTime(200 * time.Millisecond)
		if err := r.RenderVisibleObjectGroups(); err != nil {
			t.Fatalf("render objects: %v", err)
		}
		// Object 3 (collection) at (0,0)-(8,8), green at t=200ms.
		if got := r.Result.NRGBAAt(4, 4); got != colorGreen {
			t.Errorf("collection object = %v, want green", got)
		}
	})

	t.Run("rotated tile object keeps rotation while animating", func(t *testing.T) {
		_, r := loadAnimationMap(t, fsys)
		r.SetAnimationTime(0)
		if err := r.RenderVisibleObjectGroups(); err != nil {
			t.Fatalf("render objects: %v", err)
		}
		// Object 2 rotated 90° around its anchor at (16,24). The square's box
		// spans x [16,24), y [24,32) after rotation; the red frame must be drawn
		// there, while the spot above the box stays empty.
		if got := r.Result.NRGBAAt(20, 28); got != colorRed {
			t.Errorf("rotated object center = %v, want red", got)
		}
		if got := r.Result.NRGBAAt(20, 20); got.A != 0 {
			t.Errorf("above rotated box = %v, want transparent", got)
		}

		// The rotation must survive the frame swap as well.
		_, r = loadAnimationMap(t, fsys)
		r.SetAnimationTime(200 * time.Millisecond)
		if err := r.RenderVisibleObjectGroups(); err != nil {
			t.Fatalf("render objects: %v", err)
		}
		if got := r.Result.NRGBAAt(20, 28); got != colorGreen {
			t.Errorf("rotated object at t=200ms = %v, want green", got)
		}
	})
}

func TestRenderer_Animation_DisabledByDefault(t *testing.T) {
	fsys := animationTestFS(t)
	_, r := loadAnimationMap(t, fsys)
	// No SetAnimationTime: static red tile even at a time that would be green.
	if err := r.RenderVisibleLayers(); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := r.Result.NRGBAAt(4, 4); got != colorRed {
		t.Errorf("default render = %v, want static red", got)
	}

	// Re-disabling after enabling must restore static behavior.
	r.SetAnimationTime(200 * time.Millisecond)
	r.DisableAnimation()
	r.Clear()
	if err := r.RenderVisibleLayers(); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := r.Result.NRGBAAt(4, 4); got != colorRed {
		t.Errorf("after DisableAnimation = %v, want static red", got)
	}
}

// TestRenderer_Animation_ReuseAcrossTimes ensures changing the animation time
// on an existing renderer invalidates cached frame selections.
func TestRenderer_Animation_ReuseAcrossTimes(t *testing.T) {
	fsys := animationTestFS(t)
	_, r := loadAnimationMap(t, fsys)

	r.SetAnimationTime(0)
	if err := r.RenderVisibleLayers(); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := r.Result.NRGBAAt(4, 4); got != colorRed {
		t.Fatalf("t=0 = %v, want red", got)
	}

	r.Clear()
	r.SetAnimationTime(200 * time.Millisecond)
	if err := r.RenderVisibleLayers(); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := r.Result.NRGBAAt(4, 4); got != colorGreen {
		t.Fatalf("t=200ms = %v, want green", got)
	}

	r.Clear()
	r.SetAnimationTime(400 * time.Millisecond)
	if err := r.RenderVisibleLayers(); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := r.Result.NRGBAAt(4, 4); got != colorRed {
		t.Fatalf("t=400ms = %v, want red (loop wrapped)", got)
	}
}

func TestRenderer_Animation_DeterministicSameTime(t *testing.T) {
	fsys := animationTestFS(t)

	renderAt := func(at time.Duration) *image.NRGBA {
		_, r := loadAnimationMap(t, fsys)
		r.SetAnimationTime(at)
		if err := r.RenderVisibleLayers(); err != nil {
			t.Fatalf("render: %v", err)
		}
		return r.Result
	}

	a := renderAt(200 * time.Millisecond)
	b := renderAt(200 * time.Millisecond)
	c := renderAt(200*time.Millisecond + 400*time.Millisecond) // same phase, one loop later

	for y := 0; y < 40; y++ {
		for x := 0; x < 32; x++ {
			pa, pb, pc := a.NRGBAAt(x, y), b.NRGBAAt(x, y), c.NRGBAAt(x, y)
			if pa != pb {
				t.Fatalf("same time rendered differently at (%d,%d): %v vs %v", x, y, pa, pb)
			}
			if pa != pc {
				t.Fatalf("equivalent phase rendered differently at (%d,%d): %v vs %v", x, y, pa, pc)
			}
		}
	}
}

// TestRenderer_Animation_MalformedFrames covers the deterministic fallbacks:
// zero-duration-only animations pin the first frame, and frames referencing
// tiles without an image (collection) keep the placed tile.
func TestRenderer_Animation_MalformedFrames(t *testing.T) {
	atlas := image.NewNRGBA(image.Rect(0, 0, 16, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			atlas.SetNRGBA(x, y, colorRed)
			atlas.SetNRGBA(8+x, y, colorGreen)
		}
	}

	goodCollection := solidImage(image.Rect(0, 0, 8, 8), colorBlue)

	// Atlas tile 0: all-zero-duration animation -> pinned first frame (0).
	// Collection tile 0: frame points at tile 1, which has no <image> ->
	// fallback to the placed tile's own image.
	atlasTSX := `<?xml version="1.0" encoding="UTF-8"?>
<tileset name="atlas" tilewidth="8" tileheight="8" tilecount="2" columns="2">
 <image source="atlas.png" width="16" height="8"/>
 <tile id="0">
  <animation>
   <frame tileid="0" duration="0"/>
   <frame tileid="1" duration="0"/>
  </animation>
 </tile>
</tileset>`
	collectionTSX := `<?xml version="1.0" encoding="UTF-8"?>
<tileset name="collection" tilewidth="8" tileheight="8" tilecount="2" columns="1">
 <tile id="0"><image source="c0.png" width="8" height="8"/>
  <animation>
   <frame tileid="1" duration="100"/>
  </animation>
 </tile>
 <tile id="1"/>
</tileset>`
	tmx := `<?xml version="1.0" encoding="UTF-8"?>
<map version="1.2" orientation="orthogonal" renderorder="right-down" width="2" height="1" tilewidth="8" tileheight="8" infinite="0" nextlayerid="2" nextobjectid="1">
 <tileset firstgid="1" source="atlas.tsx"/>
 <tileset firstgid="3" source="collection.tsx"/>
 <layer id="1" name="tiles" width="2" height="1">
  <data encoding="csv">1,3</data>
 </layer>
</map>`

	fsys := fstest.MapFS{
		"map.tmx":        {Data: []byte(tmx)},
		"atlas.tsx":      {Data: []byte(atlasTSX)},
		"atlas.png":      {Data: pngBytes(t, atlas)},
		"collection.tsx": {Data: []byte(collectionTSX)},
		"c0.png":         {Data: pngBytes(t, goodCollection)},
	}

	_, r := loadAnimationMap(t, fsys)
	r.SetAnimationTime(5 * time.Second) // any time; both cases are time-independent
	if err := r.RenderVisibleLayers(); err != nil {
		t.Fatalf("render: %v", err)
	}

	if got := r.Result.NRGBAAt(4, 4); got != colorRed {
		t.Errorf("zero-duration animation = %v, want pinned red first frame", got)
	}
	if got := r.Result.NRGBAAt(12, 4); got != colorBlue {
		t.Errorf("frame without image = %v, want fallback blue placed tile", got)
	}
}
