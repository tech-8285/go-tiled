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
	"image/draw"

	"github.com/lafriks/go-tiled"
)

// layerState accumulates the rendering attributes inherited from enclosing
// group layers while the map's layer tree is walked. Offsets are in screen
// pixels, so they compose identically for all four map orientations.
type layerState struct {
	offsetX float64
	offsetY float64
	opacity float32
	tint    *tiled.HexColor
}

// child returns the state for a layer nested inside a layer (or group) with
// the given attributes, following Tiled's editor semantics: offsets add up,
// opacities multiply and tint colors multiply component-wise.
func (s layerState) child(offsetX, offsetY float64, opacity float32, tint *tiled.HexColor) layerState {
	return layerState{
		offsetX: s.offsetX + offsetX,
		offsetY: s.offsetY + offsetY,
		opacity: s.opacity * opacity,
		tint:    multiplyTint(s.tint, tint),
	}
}

// multiplyTint combines two tint colors the way Tiled does: component-wise
// multiplication. A nil tint is the identity (no tinting).
func multiplyTint(a, b *tiled.HexColor) *tiled.HexColor {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	ar, ag, ab, aa := a.Components()
	br, bg, bb, ba := b.Components()
	mul := func(x, y uint8) uint32 {
		return uint32(uint16(x) * uint16(y) / 0xff)
	}
	return tiled.NewHexColor(mul(ar, br), mul(ag, bg), mul(ab, bb), mul(aa, ba))
}

// tintImage returns a copy of img with every pixel multiplied by tint,
// component-wise, matching Tiled's layer tinting. The source image (which may
// be shared through the tile cache) is never modified.
func tintImage(img image.Image, tint *tiled.HexColor) image.Image {
	tr, tg, tb, ta := tint.Components()
	if tr == 0xff && tg == 0xff && tb == 0xff && ta == 0xff {
		return img
	}

	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

	mr, mg, mb, ma := uint32(tr), uint32(tg), uint32(tb), uint32(ta)
	for i := 0; i+3 < len(dst.Pix); i += 4 {
		dst.Pix[i+0] = uint8(uint32(dst.Pix[i+0]) * mr / 0xff)
		dst.Pix[i+1] = uint8(uint32(dst.Pix[i+1]) * mg / 0xff)
		dst.Pix[i+2] = uint8(uint32(dst.Pix[i+2]) * mb / 0xff)
		dst.Pix[i+3] = uint8(uint32(dst.Pix[i+3]) * ma / 0xff)
	}
	return dst
}

// colorKeyImage returns a copy of img where every pixel matching the RGB of
// the transparency color is made fully transparent.
func colorKeyImage(img image.Image, key *tiled.HexColor) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

	kr, kg, kb, _ := key.Components()
	for i := 0; i+3 < len(dst.Pix); i += 4 {
		if dst.Pix[i+0] == kr && dst.Pix[i+1] == kg && dst.Pix[i+2] == kb {
			dst.Pix[i+3] = 0
		}
	}
	return dst
}

// layerContainer is the set of child layers of a map or group, together with
// their file (render) order.
type layerContainer struct {
	layers       []*tiled.Layer
	objectGroups []*tiled.ObjectGroup
	imageLayers  []*tiled.ImageLayer
	groups       []*tiled.Group
	refs         []tiled.LayerRef
}

// RenderMap renders the entire map the way Tiled displays it: tile layers,
// object groups, image layers and group layers are composited in the exact
// order they appear in the map file, with groups nested to any depth.
// Visibility, rendering offsets, opacity and tint color of every layer and
// group are honored, and hidden layers or groups (including everything they
// contain) are skipped. Tiles and images with transparent pixels are
// alpha-composited over the layers below them.
func (r *Renderer) RenderMap() error {
	return r.renderContainer(layerContainer{
		layers:       r.m.Layers,
		objectGroups: r.m.ObjectGroups,
		imageLayers:  r.m.ImageLayers,
		groups:       r.m.Groups,
		refs:         r.m.LayersInOrder(),
	}, layerState{opacity: 1})
}

func (r *Renderer) renderContainer(c layerContainer, st layerState) error {
	for _, ref := range c.refs {
		var err error
		switch ref.Type {
		case tiled.TileLayerType:
			err = r.renderMapTileLayer(c.layers[ref.Index], st)
		case tiled.ObjectGroupLayerType:
			err = r.renderMapObjectGroup(c.objectGroups[ref.Index], st)
		case tiled.ImageLayerType:
			err = r.renderMapImageLayer(c.imageLayers[ref.Index], st)
		case tiled.GroupLayerType:
			err = r.renderMapGroup(c.groups[ref.Index], st)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// renderMapGroup renders a group layer by recursing into its children with
// the group's own visibility, offset, opacity and tint color applied.
func (r *Renderer) renderMapGroup(group *tiled.Group, st layerState) error {
	if !group.Visible {
		return nil
	}
	st = st.child(group.OffsetX, group.OffsetY, group.Opacity, group.TintColor)
	return r.renderContainer(layerContainer{
		layers:       group.Layers,
		objectGroups: group.ObjectGroups,
		imageLayers:  group.ImageLayers,
		groups:       group.Groups,
		refs:         group.LayersInOrder(),
	}, st)
}

// renderMapTileLayer renders a single tile layer with the inherited group
// state and the layer's own attributes applied.
func (r *Renderer) renderMapTileLayer(layer *tiled.Layer, st layerState) error {
	if !layer.Visible {
		return nil
	}
	st = st.child(layer.OffsetX, layer.OffsetY, layer.Opacity, layer.TintColor)
	return r.renderLayer(layer, st.offsetX, st.offsetY, st.opacity, st.tint)
}

// renderMapObjectGroup renders a single object group with the inherited
// group state and the layer's own attributes applied.
func (r *Renderer) renderMapObjectGroup(objectGroup *tiled.ObjectGroup, st layerState) error {
	if !objectGroup.Visible {
		return nil
	}
	st = st.child(objectGroup.OffsetX, objectGroup.OffsetY, objectGroup.Opacity, objectGroup.TintColor)
	for _, obj := range r.sortedObjects(objectGroup) {
		if err := r.renderObject(obj.obj, obj.sx+st.offsetX, obj.sy+st.offsetY, st.opacity, st.tint); err != nil {
			return err
		}
	}
	return nil
}

// renderMapImageLayer renders a single image layer with the inherited group
// state and the layer's own attributes applied. The image's transparency
// color key (if any) is honored, and the image is repeated across the map
// when the layer's RepeatX/RepeatY flags are set.
func (r *Renderer) renderMapImageLayer(layer *tiled.ImageLayer, st layerState) error {
	if !layer.Visible || layer.Image == nil {
		return nil
	}

	sf, err := r.open(r.m.GetFileFullPath(layer.Image.Source))
	if err != nil {
		return err
	}
	defer sf.Close()

	img, _, err := image.Decode(sf)
	if err != nil {
		return err
	}

	if layer.Image.Trans != nil {
		img = colorKeyImage(img, layer.Image.Trans)
	}

	// The deprecated X/Y attributes are in pixels and are normally zero;
	// OffsetX/OffsetY are the modern replacements.
	st = st.child(layer.OffsetX+float64(layer.X), layer.OffsetY+float64(layer.Y), layer.Opacity, layer.TintColor)

	if st.tint != nil {
		img = tintImage(img, st.tint)
	}

	var mask *image.Uniform
	if st.opacity < 1 {
		mask = image.NewUniform(color.Alpha{uint8(st.opacity * 255)})
	}

	pos := image.Pt(int(st.offsetX), int(st.offsetY))
	size := img.Bounds().Size()
	bounds := r.Result.Bounds()

	xs := []int{pos.X}
	if layer.RepeatX && size.X > 0 {
		xs = repeatPositions(pos.X, size.X, bounds.Min.X, bounds.Max.X)
	}
	ys := []int{pos.Y}
	if layer.RepeatY && size.Y > 0 {
		ys = repeatPositions(pos.Y, size.Y, bounds.Min.Y, bounds.Max.Y)
	}

	for _, y := range ys {
		for _, x := range xs {
			rect := image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x, y).Add(size)}
			if mask != nil {
				draw.DrawMask(r.Result, rect, img, img.Bounds().Min, mask, mask.Bounds().Min, draw.Over)
			} else {
				draw.Draw(r.Result, rect, img, img.Bounds().Min, draw.Over)
			}
		}
	}

	return nil
}

// repeatPositions returns every position start+k*step whose step-sized tile
// intersects [min, max), so a repeated image layer covers the whole map.
func repeatPositions(start, step, min, max int) []int {
	for start > min {
		start -= step
	}
	positions := []int{}
	for p := start; p < max; p += step {
		positions = append(positions, p)
	}
	return positions
}
