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

	"github.com/disintegration/imaging"
	tiled "github.com/lafriks/go-tiled"
)

// resizeTileImage applies a tileset's FillMode when its tile art has to be
// drawn at a size other than its native one (TileRenderSize "grid" on a tile
// layer, or a tile object whose width/height differ from the art).
//
// Per the TMX spec:
//   - "stretch" (the default): scale the image to exactly fill dst, ignoring
//     its aspect ratio.
//   - "preserve-aspect-fit": scale uniformly so the whole image fits inside
//     dst, keeping its aspect ratio, and center it. The returned image always
//     has dst's dimensions (padded with transparent pixels) so callers can
//     treat it like any other tile-sized image.
func resizeTileImage(img image.Image, dst image.Point, fillMode string) image.Image {
	src := img.Bounds().Size()
	if src.Eq(dst) {
		return img
	}

	if fillMode == tiled.FillModePreserveAspectFit {
		// Scale by the ratio that makes the image fit within dst on both
		// axes; unlike imaging.Fit this also covers the upscaling case.
		scale := min(float64(dst.X)/float64(src.X), float64(dst.Y)/float64(src.Y))
		scaledSize := image.Pt(max(1, int(float64(src.X)*scale+0.5)), max(1, int(float64(src.Y)*scale+0.5)))
		scaled := imaging.Resize(img, scaledSize.X, scaledSize.Y, imaging.NearestNeighbor)

		if scaledSize.Eq(dst) {
			return scaled
		}

		// Center the scaled art inside a dst-sized transparent canvas.
		canvas := image.NewNRGBA(image.Rectangle{Max: dst})
		offset := image.Pt((dst.X-scaledSize.X)/2, (dst.Y-scaledSize.Y)/2)
		return imaging.Paste(canvas, scaled, offset)
	}

	return imaging.Resize(img, dst.X, dst.Y, imaging.NearestNeighbor)
}

// applyLayerTileRenderSize returns the image to draw for a tile on a tile
// layer, honoring the tileset's TileRenderSize and FillMode.
//
// With TileRenderSize "tile" (the default, and the only behavior the renderer
// had before this) the art is drawn at its native cropped size and the image
// is returned unchanged. With "grid" every tile is instead rendered into the
// map's grid cell (map TileWidth x TileHeight), stretched or aspect-fitted
// according to FillMode.
func (r *Renderer) applyLayerTileRenderSize(tile *tiled.LayerTile, img image.Image) image.Image {
	if tile.Tileset.TileRenderSize != tiled.TileRenderSizeGrid {
		return img
	}

	grid := image.Pt(r.m.TileWidth, r.m.TileHeight)
	return resizeTileImage(img, grid, tile.Tileset.FillMode)
}

// tileOffset returns the tileset's drawing offset, or zero when none is
// declared. The TMX spec applies this offset both to tiles on tile layers and
// to tile objects.
func tileOffset(ts *tiled.Tileset) image.Point {
	if ts.TileOffset == nil {
		return image.Pt(0, 0)
	}
	return image.Pt(ts.TileOffset.X, ts.TileOffset.Y)
}

// objectAnchor resolves the point within a tile object's (unrotated) image
// that must align with the object's projected screen position.
//
// A tileset's ObjectAlignment overrides the orientation-dependent default
// (bottom-left for orthogonal/hexagonal/staggered maps, bottom-center for
// isometric maps); "unspecified" (the default) keeps that orientation rule.
// Points follow Tiled's 0-based alignment math: the right edge is imgSize.X
// and the bottom edge is imgSize.Y (one past the last pixel row/column).
func (r *Renderer) objectAnchor(ts *tiled.Tileset, imgSize image.Point) image.Point {
	switch ts.ObjectAlignment {
	case tiled.ObjectAlignmentTopLeft:
		return image.Pt(0, 0)
	case tiled.ObjectAlignmentTop:
		return image.Pt(imgSize.X/2, 0)
	case tiled.ObjectAlignmentTopRight:
		return image.Pt(imgSize.X, 0)
	case tiled.ObjectAlignmentLeft:
		return image.Pt(0, imgSize.Y/2)
	case tiled.ObjectAlignmentCenter:
		return image.Pt(imgSize.X/2, imgSize.Y/2)
	case tiled.ObjectAlignmentRight:
		return image.Pt(imgSize.X, imgSize.Y/2)
	case tiled.ObjectAlignmentBottomLeft:
		return image.Pt(0, imgSize.Y)
	case tiled.ObjectAlignmentBottom:
		return image.Pt(imgSize.X/2, imgSize.Y)
	case tiled.ObjectAlignmentBottomRight:
		return image.Pt(imgSize.X, imgSize.Y)
	default: // "unspecified"
		return r.engine.GetObjectAnchor(imgSize)
	}
}
