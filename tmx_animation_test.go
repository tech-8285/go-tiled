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

import "testing"

func TestTilesetTile_FrameAt(t *testing.T) {
	// Frames 10 (200ms), 11 (300ms), 12 (500ms); loop length 1000ms.
	tile := &TilesetTile{
		ID: 5,
		Animation: []*AnimationFrame{
			{TileID: 10, Duration: 200},
			{TileID: 11, Duration: 300},
			{TileID: 12, Duration: 500},
		},
	}

	tests := []struct {
		name   string
		timeMS uint64
		want   uint32
	}{
		{"start of first frame", 0, 10},
		{"inside first frame", 199, 10},
		{"boundary advances to second frame", 200, 11},
		{"inside second frame", 499, 11},
		{"boundary advances to third frame", 500, 12},
		{"last millisecond of loop", 999, 12},
		{"loop wraps back to first frame", 1000, 10},
		{"stable after several loops", 2500, 12}, // 2500 % 1000 == 500 -> third frame
		{"large time wraps identically", 1000*1234 + 200, 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tile.FrameAt(tt.timeMS); got != tt.want {
				t.Errorf("FrameAt(%d) = %d, want %d", tt.timeMS, got, tt.want)
			}
		})
	}
}

func TestTilesetTile_FrameAt_StaticAndMalformed(t *testing.T) {
	t.Run("tile without animation stays static", func(t *testing.T) {
		static := &TilesetTile{ID: 7}
		for _, timeMS := range []uint64{0, 1, 500, 1 << 40} {
			if got := static.FrameAt(timeMS); got != 7 {
				t.Errorf("FrameAt(%d) = %d, want 7", timeMS, got)
			}
		}
	})

	t.Run("nil tile resolves to zero", func(t *testing.T) {
		var nilTile *TilesetTile
		if got := nilTile.FrameAt(42); got != 0 {
			t.Errorf("nil FrameAt = %d, want 0", got)
		}
	})

	t.Run("zero-duration frames are skipped", func(t *testing.T) {
		// First frame lasts 0ms, so time 0 must already show frame 2.
		tile := &TilesetTile{
			Animation: []*AnimationFrame{
				{TileID: 20, Duration: 0},
				{TileID: 21, Duration: 100},
			},
		}
		if got := tile.FrameAt(0); got != 21 {
			t.Errorf("FrameAt(0) = %d, want 21", got)
		}
		if got := tile.FrameAt(99); got != 21 {
			t.Errorf("FrameAt(99) = %d, want 21", got)
		}
		if got := tile.FrameAt(100); got != 21 {
			t.Errorf("FrameAt(100) = %d, want 21 (loop)", got)
		}
	})

	t.Run("all zero-duration animation pins first frame", func(t *testing.T) {
		tile := &TilesetTile{
			Animation: []*AnimationFrame{
				{TileID: 30, Duration: 0},
				{TileID: 31, Duration: 0},
			},
		}
		for _, timeMS := range []uint64{0, 100, 1 << 32} {
			if got := tile.FrameAt(timeMS); got != 30 {
				t.Errorf("FrameAt(%d) = %d, want 30", timeMS, got)
			}
		}
	})

	t.Run("empty animation stays static", func(t *testing.T) {
		tile := &TilesetTile{ID: 9, Animation: []*AnimationFrame{}}
		if got := tile.FrameAt(123); got != 9 {
			t.Errorf("FrameAt(123) = %d, want 9", got)
		}
	})
}

func TestTileset_AnimatedTileID(t *testing.T) {
	ts := &Tileset{
		FirstGID:  1,
		TileCount: 12,
		Tiles: []*TilesetTile{
			{
				ID: 5,
				Animation: []*AnimationFrame{
					{TileID: 10, Duration: 100},
					{TileID: 11, Duration: 100},
				},
			},
			{
				ID: 6,
				Animation: []*AnimationFrame{
					// Frame points outside the tileset: must fall back.
					{TileID: 99, Duration: 100},
				},
			},
		},
	}

	t.Run("animated tile resolves frame", func(t *testing.T) {
		if got := ts.AnimatedTileID(5, 0); got != 10 {
			t.Errorf("AnimatedTileID(5, 0) = %d, want 10", got)
		}
		if got := ts.AnimatedTileID(5, 100); got != 11 {
			t.Errorf("AnimatedTileID(5, 100) = %d, want 11", got)
		}
		if got := ts.AnimatedTileID(5, 200); got != 10 {
			t.Errorf("AnimatedTileID(5, 200) = %d, want 10 (loop)", got)
		}
	})

	t.Run("unknown tile stays static", func(t *testing.T) {
		if got := ts.AnimatedTileID(4, 100); got != 4 {
			t.Errorf("AnimatedTileID(4, 100) = %d, want 4", got)
		}
	})

	t.Run("out-of-bounds frame falls back to placed tile", func(t *testing.T) {
		if got := ts.AnimatedTileID(6, 0); got != 6 {
			t.Errorf("AnimatedTileID(6, 0) = %d, want 6", got)
		}
	})
}
