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

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const layerOrderMap = `<?xml version="1.0" encoding="UTF-8"?>
<map version="1.10" orientation="orthogonal" renderorder="right-down" width="2" height="2" tilewidth="16" tileheight="16" infinite="0">
 <imagelayer id="1" name="bg" tintcolor="#FF0000">
  <image source="bg.png" width="16" height="16"/>
 </imagelayer>
 <layer id="2" name="ground" width="2" height="2" tintcolor="#80FF0000">
  <data encoding="csv">
0,0,
0,0
</data>
 </layer>
 <objectgroup id="3" name="objs" tintcolor="#00FF00"/>
 <group id="4" name="g" tintcolor="#0000FF">
  <layer id="5" name="inner" width="2" height="2">
   <data encoding="csv">
0,0,
0,0
</data>
  </layer>
  <imagelayer id="6" name="innerimg"/>
  <group id="7" name="nested">
   <objectgroup id="8" name="nestedobjs"/>
  </group>
 </group>
 <layer id="9" name="top" width="2" height="2">
  <data encoding="csv">
0,0,
0,0
</data>
 </layer>
</map>`

func TestMap_LayersInOrder(t *testing.T) {
	m, err := LoadReader("", strings.NewReader(layerOrderMap))
	if err != nil {
		t.Fatal(err)
	}

	expected := []LayerRef{
		{Type: ImageLayerType, Index: 0},
		{Type: TileLayerType, Index: 0},
		{Type: ObjectGroupLayerType, Index: 0},
		{Type: GroupLayerType, Index: 0},
		{Type: TileLayerType, Index: 1},
	}
	assert.Equal(t, expected, m.LayersInOrder())

	// Sanity check: the per-kind slices are still filled as before.
	assert.Len(t, m.Layers, 2)
	assert.Len(t, m.ObjectGroups, 1)
	assert.Len(t, m.ImageLayers, 1)
	assert.Len(t, m.Groups, 1)

	g := m.Groups[0]
	assert.Equal(t, []LayerRef{
		{Type: TileLayerType, Index: 0},
		{Type: ImageLayerType, Index: 0},
		{Type: GroupLayerType, Index: 0},
	}, g.LayersInOrder())

	nested := g.Groups[0]
	assert.Equal(t, []LayerRef{
		{Type: ObjectGroupLayerType, Index: 0},
	}, nested.LayersInOrder())
}

func TestMap_LayersInOrder_ProgrammaticFallback(t *testing.T) {
	// Maps built in code (no decode order recorded) fall back to the
	// historical rendering order.
	m := &Map{
		Layers:       []*Layer{{}, {}},
		ObjectGroups: []*ObjectGroup{{}},
		ImageLayers:  []*ImageLayer{{}},
		Groups:       []*Group{{}},
	}
	assert.Equal(t, []LayerRef{
		{Type: TileLayerType, Index: 0},
		{Type: TileLayerType, Index: 1},
		{Type: ObjectGroupLayerType, Index: 0},
		{Type: ImageLayerType, Index: 0},
		{Type: GroupLayerType, Index: 0},
	}, m.LayersInOrder())

	g := &Group{Layers: []*Layer{{}}, Groups: []*Group{{}}}
	assert.Equal(t, []LayerRef{
		{Type: TileLayerType, Index: 0},
		{Type: GroupLayerType, Index: 0},
	}, g.LayersInOrder())
}

func TestMap_LayerTintColor(t *testing.T) {
	m, err := LoadReader("", strings.NewReader(layerOrderMap))
	if err != nil {
		t.Fatal(err)
	}

	assertTint := func(c *HexColor, r, g, b, a uint8) {
		if !assert.NotNil(t, c) {
			return
		}
		cr, cg, cb, ca := c.Components()
		assert.Equal(t, r, cr)
		assert.Equal(t, g, cg)
		assert.Equal(t, b, cb)
		assert.Equal(t, a, ca)
	}

	assertTint(m.ImageLayers[0].TintColor, 0xff, 0x00, 0x00, 0xff)
	assertTint(m.Layers[0].TintColor, 0xff, 0x00, 0x00, 0x80)
	assertTint(m.ObjectGroups[0].TintColor, 0x00, 0xff, 0x00, 0xff)
	assertTint(m.Groups[0].TintColor, 0x00, 0x00, 0xff, 0xff)
	assert.Nil(t, m.Layers[1].TintColor)
}
