/*
Copyright (c) 2020 Lauris Bukšis <lauris@nix.lv>

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
	"encoding/xml"
)

// Group is a group layer, used to organize the layers of the map in a hierarchy.
type Group struct {
	// Unique ID of the layer.
	// Each layer that added to a map gets a unique id. Even if a layer is deleted,
	// no layer ever gets the same ID. Can not be changed in Tiled. (since Tiled 1.2)
	ID uint32 `xml:"id,attr"`
	// The name of the group layer.
	Name string `xml:"name,attr"`
	// The class of the group layer (since 1.9, defaults to "").
	Class string `xml:"class,attr"`
	// Rendering offset of the image layer in pixels. Defaults to 0. (since 0.15)
	OffsetX float64 `xml:"offsetx,attr"`
	// Rendering offset of the image layer in pixels. Defaults to 0. (since 0.15)
	OffsetY float64 `xml:"offsety,attr"`
	// The opacity of the layer as a value from 0 to 1. Defaults to 1.
	Opacity float32 `xml:"opacity,attr"`
	// Whether the layer is shown (1) or hidden (0). Defaults to 1.
	Visible bool `xml:"visible,attr"`
	// A color that is multiplied with any graphics drawn by any child layers. (optional)
	TintColor *HexColor `xml:"tintcolor,attr"`
	// The parallax x factor of the layer 0 - 1.0
	ParallaxX float32 `xml:"parallaxx,attr"`
	// The parallax y factor of the layer 0 - 1.0
	ParallaxY float32 `xml:"parallaxy,attr"`
	// The compositing mode to use for this layer. Valid values are normal,
	// multiply, multiply (legacy), screen, overlay, darken, lighten, color dodge,
	// color burn, hard light, soft light, difference, exclusion, hue, saturation,
	// color and luminosity. (since 1.12, defaults to normal)
	Mode string `xml:"mode,attr"`
	// Custom properties
	Properties Properties `xml:"properties>property"`
	// Map layers
	Layers []*Layer `xml:"layer"`
	// Map object groups
	ObjectGroups []*ObjectGroup `xml:"objectgroup"`
	// Image layers
	ImageLayers []*ImageLayer `xml:"imagelayer"`
	// Group layers
	Groups []*Group `xml:"group"`

	// layerOrder records the document order of the layer, objectgroup,
	// imagelayer and group child elements. It is populated while decoding;
	// nil for groups built in code.
	layerOrder []LayerRef
}

// LayersInOrder returns references to all child layers of the group — tile
// layers, object groups, image layers and subgroups — in the order they
// appear in the map file, which is the order Tiled renders them in. For
// groups that were constructed in code rather than decoded from a file, the
// historical order (tile layers, object groups, image layers, groups) is
// returned.
func (g *Group) LayersInOrder() []LayerRef {
	if g.layerOrder != nil {
		return g.layerOrder
	}
	return layerOrderFallback(len(g.Layers), len(g.ObjectGroups), len(g.ImageLayers), len(g.Groups))
}

// groupDecode mirrors Group for XML decoding. The layer-kind fields shadow
// the promoted aliasGroup fields (a shallower field wins over an embedded one
// with the same XML name) so that each layer element is decoded individually
// and its document position recorded in the shared order list.
type groupDecode struct {
	aliasGroup
	TileLayers tileLayerSeq   `xml:"layer"`
	ObjGroups  objectGroupSeq `xml:"objectgroup"`
	ImgLayers  imageLayerSeq  `xml:"imagelayer"`
	SubGroups  groupSeq       `xml:"group"`
}

// UnmarshalXML decodes a single XML element beginning with the given start element.
func (g *Group) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	item := aliasGroup{}
	item.SetDefaults()

	var order []LayerRef
	dec := groupDecode{aliasGroup: item}
	dec.TileLayers.order = &order
	dec.ObjGroups.order = &order
	dec.ImgLayers.order = &order
	dec.SubGroups.order = &order

	if err := d.DecodeElement(&dec, &start); err != nil {
		return err
	}

	item = dec.aliasGroup
	item.Layers = dec.TileLayers.Layers
	item.ObjectGroups = dec.ObjGroups.ObjectGroups
	item.ImageLayers = dec.ImgLayers.ImageLayers
	item.Groups = dec.SubGroups.Groups
	item.layerOrder = order

	*g = (Group)(item)

	return nil
}

// DecodeGroup decodes Group data. This includes all subgroups and the Layer
// data for each.
func (g *Group) DecodeGroup(m *Map) error {
	for i := 0; i < len(g.Groups); i++ {
		g := g.Groups[i]
		if err := g.DecodeGroup(m); err != nil {
			return err
		}
	}

	for i := 0; i < len(g.Layers); i++ {
		l := g.Layers[i]
		if err := l.DecodeLayer(m); err != nil {
			return err
		}
	}

	for _, og := range g.ObjectGroups {
		if err := og.DecodeObjectGroup(m); err != nil {
			return err
		}
	}

	return nil
}
