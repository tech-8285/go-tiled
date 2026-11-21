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
	"time"

	"github.com/lafriks/go-tiled"
)

// SetAnimationTime enables animated tile rendering and selects the frame
// visible at timeMS milliseconds into every tile animation.
//
// Animated tiles loop through their frames using each frame's duration; the
// same timeMS always produces the same image, so a screenshot taken at a fixed
// time is deterministic. The flip, rotation and placement offsets of a tile in
// a tile layer, or of a tile object in an object group, are kept from the
// original placement and applied to the selected animation frame.
//
// A negative or zero duration pins every animation to its first frame.
// Call DisableAnimation to render the static tiles as placed in the map again.
func (r *Renderer) SetAnimationTime(d time.Duration) {
	var timeMS uint64
	if d > 0 {
		timeMS = uint64(d.Milliseconds())
	}
	r.animationTimeMS = timeMS
	r.animationsEnabled = true
	r.animatedTileCache = make(map[uint32]animatedTile)
}

// DisableAnimation turns animated tile rendering off, so tiles are drawn
// statically exactly as they are placed in the map. This is the default for a
// newly created Renderer.
func (r *Renderer) DisableAnimation() {
	r.animationsEnabled = false
	r.animationTimeMS = 0
	r.animatedTileCache = make(map[uint32]animatedTile)
}

// animatedTile records the resolved frame tile for a placed global tile ID:
// frameID is the tileset-local ID of the tile whose image must be drawn, and
// animated reports whether the placed tile actually carries an animation.
type animatedTile struct {
	frameID  uint32
	animated bool
}

// resolveAnimatedTile maps a placed tile to the tileset-local tile ID whose
// image should be drawn at the renderer's current animation time. Tiles
// without an animation resolve to themselves. The result is cached per global
// tile ID because frame selection depends only on the tile and the time, and
// the cache is rebuilt whenever the animation time changes.
func (r *Renderer) resolveAnimatedTile(tile *tiled.LayerTile) uint32 {
	gid := tile.Tileset.FirstGID + tile.ID
	if entry, ok := r.animatedTileCache[gid]; ok {
		if entry.animated {
			return entry.frameID
		}
		return tile.ID
	}

	entry := animatedTile{frameID: tile.ID}
	if r.animationsEnabled {
		if tsTile, err := tile.Tileset.GetTilesetTile(tile.ID); err == nil && len(tsTile.Animation) > 0 {
			frameID := tsTile.FrameAt(r.animationTimeMS)
			if r.frameTileDrawable(tile.Tileset, frameID) {
				entry.frameID = frameID
				entry.animated = true
			}
		}
	}
	r.animatedTileCache[gid] = entry
	return entry.frameID
}

// frameTileDrawable checks that an animation frame references a tile whose
// image the renderer can actually load, so malformed animations fall back to
// the placed tile instead of failing the whole render.
func (r *Renderer) frameTileDrawable(ts *tiled.Tileset, frameID uint32) bool {
	if ts.TileCount > 0 && frameID >= uint32(ts.TileCount) {
		return false
	}

	// Image-collection tilesets only have image entries for tiles that define
	// their own <image>; a frame pointing at any other local ID has nothing to
	// draw.
	if ts.Image == nil {
		for _, t := range ts.Tiles {
			if t.ID == frameID {
				return t.Image != nil
			}
		}
		return false
	}

	return true
}
