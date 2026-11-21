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
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/lafriks/go-tiled"
)

// twoColorTile renders a 16x16 tile whose left half is red and right half is
// blue, a pattern asymmetric enough to expose flip/offset mistakes.
func twoColorTile() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	red := color.NRGBA{255, 0, 0, 255}
	blue := color.NRGBA{0, 0, 255, 255}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if x < 8 {
				img.SetNRGBA(x, y, red)
			} else {
				img.SetNRGBA(x, y, blue)
			}
		}
	}
	return img
}

func pngBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

// The map rendered by both the atlas and image-collection variants: a 2x2
// orthogonal map (16px cells) with one layer tile at cell (0,0) and one
// 16x16 tile object anchored into cell (1,1). Both tilesets declare the same
// TileOffset, ObjectAlignment and TileRenderSize so the outputs must match
// pixel-for-pixel.
const parityMapTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<map version="1.9" orientation="orthogonal" renderorder="right-down" width="2" height="2" tilewidth="16" tileheight="16" infinite="0">
 <tileset firstgid="1" source="%s"/>
 <layer id="1" name="Layer" width="2" height="2">
  <data encoding="csv">1,0,
0,0</data>
 </layer>
 <objectgroup id="2" name="Objects">
  <object id="1" gid="1" x="16" y="32" width="16" height="16"/>
 </objectgroup>
</map>`

const atlasTSX = `<?xml version="1.0" encoding="UTF-8"?>
<tileset version="1.9" name="atlas" tilewidth="16" tileheight="16" tilecount="2" columns="2"
 objectalignment="center" tilerendersize="grid" fillmode="preserve-aspect-fit">
 <tileoffset x="2" y="-1"/>
 <image source="atlas.png" width="32" height="16"/>
</tileset>`

const collectionTSX = `<?xml version="1.0" encoding="UTF-8"?>
<tileset version="1.9" name="collection" tilewidth="16" tileheight="16" tilecount="2" columns="1"
 objectalignment="center" tilerendersize="grid" fillmode="preserve-aspect-fit">
 <tileoffset x="2" y="-1"/>
 <tile id="0">
  <image source="tile0.png" width="16" height="16"/>
 </tile>
 <tile id="1">
  <image source="tile1.png" width="16" height="16"/>
 </tile>
</tileset>`

// TestRenderer_AtlasAndImageCollectionParity renders the same logical map
// once against a whole-image atlas tileset and once against an image
// collection (per-tile images) and requires identical pixels, proving the
// tileset properties are honored consistently regardless of image source.
func TestRenderer_AtlasAndImageCollectionParity(t *testing.T) {
	tile := twoColorTile()

	// The atlas places two copies of the tile side by side.
	atlas := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 32; x++ {
			atlas.Set(x, y, tile.At(x%16, y))
		}
	}

	fsys := fstest.MapFS{
		"atlas.map":      {Data: []byte(fmt.Sprintf(parityMapTemplate, "atlas.tsx"))},
		"atlas.tsx":      {Data: []byte(atlasTSX)},
		"atlas.png":      {Data: pngBytes(t, atlas)},
		"collection.map": {Data: []byte(fmt.Sprintf(parityMapTemplate, "collection.tsx"))},
		"collection.tsx": {Data: []byte(collectionTSX)},
		"tile0.png":      {Data: pngBytes(t, tile)},
		"tile1.png":      {Data: pngBytes(t, tile)},
	}

	atlasResult := renderParityMap(t, fsys, "atlas.map")
	collectionResult := renderParityMap(t, fsys, "collection.map")

	atlasBounds := atlasResult.Bounds()
	if got := collectionResult.Bounds(); got != atlasBounds {
		t.Fatalf("result bounds differ: atlas=%v collection=%v", atlasBounds, got)
	}

	for y := atlasBounds.Min.Y; y < atlasBounds.Max.Y; y++ {
		for x := atlasBounds.Min.X; x < atlasBounds.Max.X; x++ {
			if got, want := collectionResult.NRGBAAt(x, y), atlasResult.NRGBAAt(x, y); got != want {
				t.Fatalf("pixel (%d,%d) = %v, want %v (atlas)", x, y, got, want)
			}
		}
	}

	// Sanity-check that something actually rendered and the layer tile's
	// tileoffset shifted it to (2,-1): its first visible pixel is at (2,0).
	if c := atlasResult.NRGBAAt(2, 0); c.A == 0 {
		t.Errorf("layer tile top visible pixel (2,0) is transparent, want the tile's art")
	}
}

func renderParityMap(t *testing.T, fsys fstest.MapFS, name string) *image.NRGBA {
	t.Helper()

	m, err := tiled.LoadFile(name, tiled.WithFileSystem(fsys))
	if err != nil {
		t.Fatalf("LoadFile(%s): %v", name, err)
	}

	renderer, err := NewRendererWithFileSystem(m, fsys)
	if err != nil {
		t.Fatalf("NewRendererWithFileSystem(%s): %v", name, err)
	}

	if err := renderer.RenderVisibleLayers(); err != nil {
		t.Fatalf("RenderVisibleLayers(%s): %v", name, err)
	}
	if err := renderer.RenderVisibleObjectGroups(); err != nil {
		t.Fatalf("RenderVisibleObjectGroups(%s): %v", name, err)
	}

	return renderer.Result
}
