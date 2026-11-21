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

package tiled

// FrameAt returns the local tile ID of the animation frame that is visible
// at timeMS milliseconds into the animation.
//
// Frames are advanced according to each frame's Duration (in milliseconds)
// and the sequence loops once the total duration of all frames has elapsed.
// The result is a pure function of timeMS, so the same instant always yields
// the same frame, including after arbitrary numbers of loops.
//
// Edge cases are handled deterministically:
//   - A tile without an animation resolves to its own tile ID, i.e. it stays
//     static.
//   - A frame with a zero duration is skipped while walking the timeline.
//   - If every frame has a zero duration (or the animation is otherwise
//     unplayable), the first frame is returned.
//   - A nil tile resolves to tile ID 0.
func (t *TilesetTile) FrameAt(timeMS uint64) uint32 {
	if t == nil {
		return 0
	}
	if len(t.Animation) == 0 {
		return t.ID
	}

	var total uint64
	for _, frame := range t.Animation {
		total += uint64(frame.Duration)
	}

	// No playable duration: pin to the first frame.
	if total == 0 {
		return t.Animation[0].TileID
	}

	elapsed := timeMS % total
	for _, frame := range t.Animation {
		duration := uint64(frame.Duration)
		if duration == 0 {
			// Zero-duration frames occupy no time on the timeline and are
			// never visible.
			continue
		}
		if elapsed < duration {
			return frame.TileID
		}
		elapsed -= duration
	}

	// Unreachable for a well-formed timeline, but keep the result defined
	// rather than returning a zero value that may mean a different tile.
	return t.Animation[len(t.Animation)-1].TileID
}

// AnimatedTileID resolves which tileset-local tile ID should be drawn in place
// of tileID at timeMS milliseconds into the animation.
//
// A tile without an animation, an unknown tile ID, or an animation whose
// selected frame references a tile outside the tileset all resolve to the
// original tileID, so callers always get a drawable, stable result.
func (ts *Tileset) AnimatedTileID(tileID uint32, timeMS uint64) uint32 {
	tile, err := ts.GetTilesetTile(tileID)
	if err != nil || len(tile.Animation) == 0 {
		return tileID
	}

	frameID := tile.FrameAt(timeMS)
	if ts.TileCount > 0 && frameID >= uint32(ts.TileCount) {
		return tileID
	}
	return frameID
}
