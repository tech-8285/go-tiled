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
	"image"
	"image/color"
	"testing"

	"github.com/lafriks/go-tiled"
)

// solidImage returns a w*h NRGBA image entirely filled with c, so individual
// pixels can be identified by color in the render output.
func solidImage(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestResizeTileImage_Stretch(t *testing.T) {
	red := solidImage(10, 5, color.NRGBA{255, 0, 0, 255})

	got := resizeTileImage(red, image.Pt(20, 20), tiled.FillModeStretch)
	if size := got.Bounds().Size(); size != image.Pt(20, 20) {
		t.Fatalf("stretch size = %v, want (20,20)", size)
	}
	if c := got.At(19, 19); c != (color.NRGBA{255, 0, 0, 255}) {
		t.Errorf("stretched pixel = %v, want opaque red", c)
	}
}

// TestResizeTileImage_PreserveAspectFit covers both the padding (image keeps
// its aspect ratio and is centered rather than stretched) and the upscaling
// path, since imaging.Fit only downscales.
func TestResizeTileImage_PreserveAspectFit(t *testing.T) {
	red := solidImage(10, 5, color.NRGBA{255, 0, 0, 255})
	dst := image.Pt(20, 20)

	got := resizeTileImage(red, dst, tiled.FillModePreserveAspectFit)
	if size := got.Bounds().Size(); size != dst {
		t.Fatalf("fit canvas size = %v, want %v", size, dst)
	}

	// Scale 0.5 -> 20x10, centered vertically at y = 5..14. The bands above
	// and below stay transparent; the middle band is opaque red.
	if c := got.At(10, 0); c != (color.NRGBA{0, 0, 0, 0}) {
		t.Errorf("top padding = %v, want transparent", c)
	}
	if c := got.At(10, 19); c != (color.NRGBA{0, 0, 0, 0}) {
		t.Errorf("bottom padding = %v, want transparent", c)
	}
	if c := got.At(10, 10); c != (color.NRGBA{255, 0, 0, 255}) {
		t.Errorf("fitted content = %v, want opaque red", c)
	}

	// Upscaling: a 5x2 image must grow (not stay native) to fit a 20x20 box.
	small := solidImage(5, 2, color.NRGBA{0, 255, 0, 255})
	up := resizeTileImage(small, dst, tiled.FillModePreserveAspectFit)
	if c := up.At(10, 10); c != (color.NRGBA{0, 255, 0, 255}) {
		t.Errorf("upscaled fit content = %v, want opaque green", c)
	}
	if c := up.At(10, 0); c != (color.NRGBA{0, 0, 0, 0}) {
		t.Errorf("upscaled fit padding = %v, want transparent", c)
	}
}

func TestResizeTileImage_NativeSizeUnchanged(t *testing.T) {
	red := solidImage(8, 8, color.NRGBA{255, 0, 0, 255})
	for _, mode := range []string{tiled.FillModeStretch, tiled.FillModePreserveAspectFit} {
		if got := resizeTileImage(red, image.Pt(8, 8), mode); got != image.Image(red) {
			t.Errorf("fill mode %q returned a new image at native size, want the original untouched", mode)
		}
	}
}

func TestTileOffset(t *testing.T) {
	if got := tileOffset(&tiled.Tileset{}); got != image.Pt(0, 0) {
		t.Errorf("nil TileOffset = %v, want zero point", got)
	}

	ts := &tiled.Tileset{TileOffset: &tiled.TilesetTileOffset{X: -3, Y: 7}}
	if got := tileOffset(ts); got != image.Pt(-3, 7) {
		t.Errorf("declared TileOffset = %v, want (-3,7)", got)
	}
}

// TestObjectAnchor_ExplicitAlignment checks every non-default ObjectAlignment
// value against its spec'd anchor point on a 10x6 image. Edges are 0-based:
// right = 10 and bottom = 6 (one past the last pixel row/column).
func TestObjectAnchor_ExplicitAlignment(t *testing.T) {
	r := &Renderer{engine: &OrthogonalRendererEngine{}}
	size := image.Pt(10, 6)

	tests := []struct {
		alignment string
		want      image.Point
	}{
		{tiled.ObjectAlignmentTopLeft, image.Pt(0, 0)},
		{tiled.ObjectAlignmentTop, image.Pt(5, 0)},
		{tiled.ObjectAlignmentTopRight, image.Pt(10, 0)},
		{tiled.ObjectAlignmentLeft, image.Pt(0, 3)},
		{tiled.ObjectAlignmentCenter, image.Pt(5, 3)},
		{tiled.ObjectAlignmentRight, image.Pt(10, 3)},
		{tiled.ObjectAlignmentBottomLeft, image.Pt(0, 6)},
		{tiled.ObjectAlignmentBottom, image.Pt(5, 6)},
		{tiled.ObjectAlignmentBottomRight, image.Pt(10, 6)},
	}

	for _, tt := range tests {
		t.Run(tt.alignment, func(t *testing.T) {
			ts := &tiled.Tileset{ObjectAlignment: tt.alignment}
			if got := r.objectAnchor(ts, size); got != tt.want {
				t.Errorf("objectAnchor(%q, %v) = %v, want %v", tt.alignment, size, got, tt.want)
			}
		})
	}
}

// TestObjectAnchor_UnspecifiedFallsBackToOrientation verifies "unspecified"
// keeps the pre-existing, orientation-dependent semantics: bottom-left for
// orthogonal, bottom-center for isometric.
func TestObjectAnchor_UnspecifiedFallsBackToOrientation(t *testing.T) {
	size := image.Pt(10, 6)
	ts := &tiled.Tileset{ObjectAlignment: tiled.ObjectAlignmentUnspecified}

	ortho := &Renderer{engine: &OrthogonalRendererEngine{}}
	if got := ortho.objectAnchor(ts, size); got != image.Pt(0, 6) {
		t.Errorf("orthogonal default anchor = %v, want (0,6)", got)
	}

	iso := &Renderer{engine: &IsometricRendererEngine{}}
	if got := iso.objectAnchor(ts, size); got != image.Pt(5, 6) {
		t.Errorf("isometric default anchor = %v, want (5,6)", got)
	}
}

// newTestRenderer builds a bare orthogonal renderer whose single tile is the
// given image, bypassing file loading by pre-populating the tile cache.
func newTestRenderer(t *testing.T, ts *tiled.Tileset, tileImage image.Image) *Renderer {
	t.Helper()

	m := &tiled.Map{
		Orientation: "orthogonal",
		Width:       2,
		Height:      2,
		TileWidth:   32,
		TileHeight:  32,
		Tilesets:    []*tiled.Tileset{ts},
	}

	r, err := NewRenderer(m)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	r.tileCache[ts.FirstGID] = tileImage
	return r
}

// TestRenderer_LayerTile_TileRenderSizeGrid renders one grid-render-size tile
// whose native art is smaller than the 32x32 cell and checks the whole cell is
// painted (stretch), while a "tile"-render-size tileset keeps native sizing.
func TestRenderer_LayerTile_TileRenderSizeGrid(t *testing.T) {
	red := color.NRGBA{255, 0, 0, 255}

	gridTS := &tiled.Tileset{
		FirstGID:       1,
		TileWidth:      16,
		TileHeight:     16,
		TileRenderSize: tiled.TileRenderSizeGrid,
		FillMode:       tiled.FillModeStretch,
	}
	tile := &tiled.LayerTile{ID: 0, Tileset: gridTS}
	r := newTestRenderer(t, gridTS, solidImage(16, 16, red))

	got := r.applyLayerTileRenderSize(tile, r.tileCache[1])
	if size := got.Bounds().Size(); size != image.Pt(32, 32) {
		t.Fatalf("grid render size = %v, want map cell (32,32)", size)
	}

	// Default "tile" render size must leave the image exactly as before.
	tileTS := &tiled.Tileset{FirstGID: 1, TileWidth: 16, TileHeight: 16, TileRenderSize: tiled.TileRenderSizeTile}
	native := solidImage(16, 16, red)
	r2 := newTestRenderer(t, tileTS, native)
	if got := r2.applyLayerTileRenderSize(&tiled.LayerTile{ID: 0, Tileset: tileTS}, native); got != image.Image(native) {
		t.Error("tile render size changed the native image, want it untouched")
	}
}

// TestRenderer_LayerTile_TileOffset verifies the tileset drawing offset is
// added on top of the normal grid-cell placement for tile layers.
func TestRenderer_LayerTile_TileOffset(t *testing.T) {
	red := color.NRGBA{255, 0, 0, 255}
	ts := &tiled.Tileset{
		FirstGID:   1,
		TileWidth:  32,
		TileHeight: 32,
		TileOffset: &tiled.TilesetTileOffset{X: 4, Y: 8},
	}
	r := newTestRenderer(t, ts, solidImage(32, 32, red))

	// A single real tile in cell (0,0), followed by nil entries; the layer's
	// unexported map reference is irrelevant to _renderLayer (it indexes r.m).
	layer := &tiled.Layer{Opacity: 1, Tiles: []*tiled.LayerTile{
		{ID: 0, Tileset: ts},
		tiled.NilLayerTile, tiled.NilLayerTile, tiled.NilLayerTile,
	}}
	if err := r._renderLayer(layer); err != nil {
		t.Fatalf("_renderLayer: %v", err)
	}

	// Without the offset cell (0,0) would cover (0,0)-(31,31); shifted by
	// (4,8) its top-left pixel lands at (4,8).
	if c := r.Result.NRGBAAt(4, 8); c != red {
		t.Errorf("offset tile top-left (4,8) = %v, want red", c)
	}
	// One pixel left of the shifted tile stays transparent.
	if c := r.Result.NRGBAAt(3, 8); c != (color.NRGBA{0, 0, 0, 0}) {
		t.Errorf("pixel (3,8) = %v, want transparent", c)
	}
	// The unshifted top-left corner stays transparent.
	if c := r.Result.NRGBAAt(0, 0); c != (color.NRGBA{0, 0, 0, 0}) {
		t.Errorf("pixel (0,0) = %v, want transparent", c)
	}
}

// TestRenderer_TileObject_AlignmentAndOffset renders one tile object and
// checks its final drawn rectangle against each ObjectAlignment plus the
// tileset TileOffset, without rotation.
func TestRenderer_TileObject_AlignmentAndOffset(t *testing.T) {
	red := color.NRGBA{255, 0, 0, 255}
	const imgW, imgH = 10, 6

	tests := []struct {
		name      string
		alignment string
		// anchor point of the 10x6 image that sits on the object position
		anchor image.Point
	}{
		{"unspecified is bottom-left", tiled.ObjectAlignmentUnspecified, image.Pt(0, imgH)},
		{"topleft", tiled.ObjectAlignmentTopLeft, image.Pt(0, 0)},
		{"center", tiled.ObjectAlignmentCenter, image.Pt(imgW/2, imgH/2)},
		{"bottomright", tiled.ObjectAlignmentBottomRight, image.Pt(imgW, imgH)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &tiled.Tileset{
				FirstGID:        1,
				TileWidth:       imgW,
				TileHeight:      imgH,
				ObjectAlignment: tt.alignment,
				TileOffset:      &tiled.TilesetTileOffset{X: 2, Y: 3},
			}
			r := newTestRenderer(t, ts, solidImage(imgW, imgH, red))

			const posX, posY = 40, 50
			obj := &tiled.Object{ID: 1, GID: 1, Visible: true, X: posX, Y: posY, Width: imgW, Height: imgH}
			og := &tiled.ObjectGroup{Opacity: 1, Objects: []*tiled.Object{obj}}
			if err := r._renderObjectGroup(og); err != nil {
				t.Fatalf("_renderObjectGroup: %v", err)
			}

			wantMinX := posX - tt.anchor.X + ts.TileOffset.X
			wantMinY := posY - tt.anchor.Y + ts.TileOffset.Y

			if c := r.Result.NRGBAAt(wantMinX, wantMinY); c != red {
				t.Errorf("top-left of drawn object (%d,%d) = %v, want red", wantMinX, wantMinY, c)
			}
			if c := r.Result.NRGBAAt(wantMinX+imgW-1, wantMinY+imgH-1); c != red {
				t.Errorf("bottom-right of drawn object = %v, want red", c)
			}
			if c := r.Result.NRGBAAt(wantMinX-1, wantMinY); c != (color.NRGBA{0, 0, 0, 0}) {
				t.Errorf("pixel just left of object = %v, want transparent", c)
			}
		})
	}
}

// TestRenderer_TileObject_ZeroSizeFallsBackToTilesetSize covers objects
// without explicit width/height: they must render at the tileset's tile size.
func TestRenderer_TileObject_ZeroSizeFallsBackToTilesetSize(t *testing.T) {
	red := color.NRGBA{255, 0, 0, 255}
	ts := &tiled.Tileset{FirstGID: 1, TileWidth: 12, TileHeight: 8}
	r := newTestRenderer(t, ts, solidImage(12, 8, red))

	// Bottom-left anchor: image's (0,8) lands on object position (20,30),
	// so it spans (20,22)-(31,29).
	obj := &tiled.Object{ID: 1, GID: 1, Visible: true, X: 20, Y: 30}
	og := &tiled.ObjectGroup{Opacity: 1, Objects: []*tiled.Object{obj}}
	if err := r._renderObjectGroup(og); err != nil {
		t.Fatalf("_renderObjectGroup: %v", err)
	}

	if c := r.Result.NRGBAAt(20, 22); c != red {
		t.Errorf("object top-left (20,22) = %v, want red", c)
	}
	if c := r.Result.NRGBAAt(31, 29); c != red {
		t.Errorf("object bottom-right (31,29) = %v, want red", c)
	}
	if c := r.Result.NRGBAAt(32, 29); c != (color.NRGBA{0, 0, 0, 0}) {
		t.Errorf("pixel (32,29) = %v, want transparent", c)
	}
}

// TestRenderer_TileObject_PreserveAspectFit makes sure FillMode is honored on
// tile objects too: a wide image fitted into a square object is centered with
// transparent padding instead of being stretched.
func TestRenderer_TileObject_PreserveAspectFit(t *testing.T) {
	red := color.NRGBA{255, 0, 0, 255}
	ts := &tiled.Tileset{
		FirstGID:   1,
		TileWidth:  20,
		TileHeight: 10,
		FillMode:   tiled.FillModePreserveAspectFit,
	}
	r := newTestRenderer(t, ts, solidImage(20, 10, red))

	// Object is a 20x20 square; 2:1 art fits as 20x10, centered (y offset 5).
	// Bottom-left anchor of the full 20x20 canvas sits on (10,40), so the
	// canvas spans (10,20)-(29,39) and the fitted art spans y 25..34.
	obj := &tiled.Object{ID: 1, GID: 1, Visible: true, X: 10, Y: 40, Width: 20, Height: 20}
	og := &tiled.ObjectGroup{Opacity: 1, Objects: []*tiled.Object{obj}}
	if err := r._renderObjectGroup(og); err != nil {
		t.Fatalf("_renderObjectGroup: %v", err)
	}

	if c := r.Result.NRGBAAt(10, 20); c != (color.NRGBA{0, 0, 0, 0}) {
		t.Errorf("padding above fitted art = %v, want transparent", c)
	}
	if c := r.Result.NRGBAAt(10, 30); c != red {
		t.Errorf("fitted art center = %v, want red", c)
	}
	if c := r.Result.NRGBAAt(10, 39); c != (color.NRGBA{0, 0, 0, 0}) {
		t.Errorf("padding below fitted art = %v, want transparent", c)
	}
}

// TestRenderer_TileObject_FlipFlags verifies transformation flags carried in
// an object's GID still take effect alongside the new sizing/alignment logic.
func TestRenderer_TileObject_FlipFlags(t *testing.T) {
	// Left half red, right half blue: an H-flip swaps the halves.
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	red := color.NRGBA{255, 0, 0, 255}
	blue := color.NRGBA{0, 0, 255, 255}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if x < 5 {
				img.SetNRGBA(x, y, red)
			} else {
				img.SetNRGBA(x, y, blue)
			}
		}
	}

	ts := &tiled.Tileset{FirstGID: 1, TileWidth: 10, TileHeight: 10}
	r := newTestRenderer(t, ts, img)

	// Horizontal flip bit (0x80000000) OR-ed onto GID 1.
	obj := &tiled.Object{ID: 1, GID: 1 | 0x80000000, Visible: true, X: 0, Y: 10, Width: 10, Height: 10}
	og := &tiled.ObjectGroup{Opacity: 1, Objects: []*tiled.Object{obj}}
	if err := r._renderObjectGroup(og); err != nil {
		t.Fatalf("_renderObjectGroup: %v", err)
	}

	// Bottom-left anchor: spans (0,0)-(9,9). After H-flip the red half is on
	// the right and blue on the left.
	if c := r.Result.NRGBAAt(0, 5); c != blue {
		t.Errorf("flipped left edge = %v, want blue", c)
	}
	if c := r.Result.NRGBAAt(9, 5); c != red {
		t.Errorf("flipped right edge = %v, want red", c)
	}
}
