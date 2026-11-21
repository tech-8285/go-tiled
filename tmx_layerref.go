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
	"encoding/xml"
)

// LayerType identifies the kind of layer a LayerRef refers to.
type LayerType int

const (
	// TileLayerType is a tile layer (Layer).
	TileLayerType LayerType = iota
	// ObjectGroupLayerType is an object group (ObjectGroup).
	ObjectGroupLayerType
	// ImageLayerType is an image layer (ImageLayer).
	ImageLayerType
	// GroupLayerType is a group layer (Group).
	GroupLayerType
)

// LayerRef references a single layer of a map or group in the order it
// appears in the map file, which is the order Tiled renders the layers in.
// Index is the layer's position in the corresponding slice (Layers,
// ObjectGroups, ImageLayers or Groups) of the parent map or group.
type LayerRef struct {
	// The kind of layer referenced.
	Type LayerType
	// Position in the parent's slice for this layer kind.
	Index int
}

// The tileLayerSeq, objectGroupSeq, imageLayerSeq and groupSeq types below
// are used while XML-decoding a map or group: the standard decoder fills each
// layer-kind slice separately, which loses the relative order of the layer
// elements in the file. These wrappers decode one element at a time and
// append a LayerRef to a shared, per-parent order list so the original
// document order can be replayed at render time.

type tileLayerSeq struct {
	order  *[]LayerRef
	Layers []*Layer
}

// UnmarshalXML decodes a single layer element and records its position.
func (s *tileLayerSeq) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	l := &Layer{}
	if err := l.UnmarshalXML(d, start); err != nil {
		return err
	}
	s.Layers = append(s.Layers, l)
	if s.order != nil {
		*s.order = append(*s.order, LayerRef{Type: TileLayerType, Index: len(s.Layers) - 1})
	}
	return nil
}

type objectGroupSeq struct {
	order        *[]LayerRef
	ObjectGroups []*ObjectGroup
}

// UnmarshalXML decodes a single objectgroup element and records its position.
func (s *objectGroupSeq) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	g := &ObjectGroup{}
	if err := g.UnmarshalXML(d, start); err != nil {
		return err
	}
	s.ObjectGroups = append(s.ObjectGroups, g)
	if s.order != nil {
		*s.order = append(*s.order, LayerRef{Type: ObjectGroupLayerType, Index: len(s.ObjectGroups) - 1})
	}
	return nil
}

type imageLayerSeq struct {
	order       *[]LayerRef
	ImageLayers []*ImageLayer
}

// UnmarshalXML decodes a single imagelayer element and records its position.
func (s *imageLayerSeq) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	l := &ImageLayer{}
	if err := l.UnmarshalXML(d, start); err != nil {
		return err
	}
	s.ImageLayers = append(s.ImageLayers, l)
	if s.order != nil {
		*s.order = append(*s.order, LayerRef{Type: ImageLayerType, Index: len(s.ImageLayers) - 1})
	}
	return nil
}

type groupSeq struct {
	order  *[]LayerRef
	Groups []*Group
}

// UnmarshalXML decodes a single group element and records its position.
func (s *groupSeq) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	g := &Group{}
	if err := g.UnmarshalXML(d, start); err != nil {
		return err
	}
	s.Groups = append(s.Groups, g)
	if s.order != nil {
		*s.order = append(*s.order, LayerRef{Type: GroupLayerType, Index: len(s.Groups) - 1})
	}
	return nil
}

// layerOrderFallback builds the historical rendering order (tile layers,
// object groups, image layers, then groups) for maps and groups that were
// constructed programmatically instead of decoded from a file, so no
// document order was recorded.
func layerOrderFallback(layers, objectGroups, imageLayers, groups int) []LayerRef {
	refs := make([]LayerRef, 0, layers+objectGroups+imageLayers+groups)
	for i := 0; i < layers; i++ {
		refs = append(refs, LayerRef{Type: TileLayerType, Index: i})
	}
	for i := 0; i < objectGroups; i++ {
		refs = append(refs, LayerRef{Type: ObjectGroupLayerType, Index: i})
	}
	for i := 0; i < imageLayers; i++ {
		refs = append(refs, LayerRef{Type: ImageLayerType, Index: i})
	}
	for i := 0; i < groups; i++ {
		refs = append(refs, LayerRef{Type: GroupLayerType, Index: i})
	}
	return refs
}
