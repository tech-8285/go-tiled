// Tool to convert a TMX file to an image.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lafriks/go-tiled"
	"github.com/lafriks/go-tiled/render"
)

func main() {
	// When omitted, animated tiles are rendered statically as placed in the map.
	animationTime := flag.Duration("time", 0, "point in time of the tile animations to render (e.g. 500ms); animated tiles stay static when this flag is not provided")
	flag.Parse()

	filename := flag.Arg(0)
	img := flag.Arg(1)
	if img == "" {
		img = "map.png"
	}

	m, err := tiled.LoadFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	}

	rend, err := render.NewRenderer(m)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Whether -time was given at all; its zero value is indistinguishable from
	// "not provided" via flag.Duration alone, and only the latter must keep
	// animated tiles static.
	timeSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "time" {
			timeSet = true
		}
	})
	if timeSet {
		rend.SetAnimationTime(*animationTime)
	}

	if err = rend.RenderVisibleLayers(); err != nil {
		fmt.Println(err)
		return
	}
	// rend.RenderLayer(1)

	w, err := os.Create(img)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer w.Close()
	if err := rend.SaveAsPng(w); err != nil {
		fmt.Println(err)
		return
	}
}
