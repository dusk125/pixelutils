package packer

import (
	"fmt"
	"sort"

	"golang.org/x/image/colornames"

	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"

	"github.com/faiface/pixel"
)

type Packer struct {
	emptySpaces  []pixel.Rect
	filledSpaces []pixel.Rect
	canvas       *pixelgl.Canvas
	flags        uint8
	im           *imdraw.IMDraw
}

const (
	PackerAllowGrowth uint8 = 1 << iota
	PackerDebugDraw
)

func NewPacker(width, height int, flags uint8) *Packer {
	packer := &Packer{
		flags: flags,
	}

	packer.canvas = pixelgl.NewCanvas(pixel.R(0, 0, float64(width), float64(height)))

	if packer.hasFlag(PackerDebugDraw) {
		packer.im = imdraw.New(nil)
		packer.im.Color = colornames.Red
		packer.im.Push(packer.canvas.Bounds().Min, packer.canvas.Bounds().Max)
		packer.im.Rectangle(5)
	}

	packer.emptySpaces = make([]pixel.Rect, 1)
	packer.emptySpaces[0] = packer.canvas.Bounds()
	packer.filledSpaces = make([]pixel.Rect, 0)

	return packer
}

func (packer *Packer) MakeTriangles(t pixel.Triangles) pixel.TargetTriangles {
	return packer.canvas.MakeTriangles(t)
}

func (packer *Packer) remove(i int) (removed pixel.Rect) {
	removed = packer.emptySpaces[i]
	packer.emptySpaces = append(packer.emptySpaces[:i], packer.emptySpaces[i+1:]...)
	return
}

func (packer Packer) hasFlag(flag uint8) bool {
	return packer.flags&flag != 0
}

func (packer *Packer) Insert(image *pixel.Sprite) (pos pixel.Vec, err error) {
	candidateIndex := 0
	if len(packer.emptySpaces) > 1 {
		found := false
		bounds := image.Picture().Bounds()
		for i, space := range packer.emptySpaces {
			if bounds.W() <= space.W() && bounds.H() <= space.H() {
				candidateIndex = i
				found = true
				break
			}
		}

		if !found {
			return pixel.ZV, fmt.Errorf("Couldn't find an empty space")
		}
	}

	space := packer.remove(candidateIndex)
	packer.filledSpaces = append(packer.filledSpaces, space)

	s := packer.split(image.Picture().Bounds(), space)

	if s.count == -1 {
		return pixel.ZV, fmt.Errorf("Failed to split the space")
	}

	if s.count == 2 {
		packer.emptySpaces = append(packer.emptySpaces, s.bigger)
	}
	packer.emptySpaces = append(packer.emptySpaces, s.smaller)

	sort.Sort(spaceSorter{packer.emptySpaces})

	if packer.hasFlag(PackerDebugDraw) {
		packer.im.Reset()
		packer.im.Clear()

		packer.im.Color = colornames.Black
		for _, item := range packer.filledSpaces {
			packer.im.Push(item.Min, item.Max)
			packer.im.Rectangle(5)
		}
		packer.im.Color = colornames.Red
		for _, item := range packer.emptySpaces {
			packer.im.Push(item.Min, item.Max)
			packer.im.Rectangle(5)
		}
	}

	image.Draw(packer.canvas, pixel.IM.Moved(space.Min).Moved(image.Picture().Bounds().Center()))

	return space.Min, nil
}

func (packer Packer) Center() pixel.Vec {
	return packer.canvas.Bounds().Center()
}

func (packer Packer) Draw(t pixel.Target, matrix pixel.Matrix) {
	packer.canvas.Draw(t, matrix)

	if packer.hasFlag(PackerDebugDraw) {
		packer.im.Draw(t)
	}
}

func rect(x, y, w, h float64) pixel.Rect {
	return pixel.R(x, y, x+w, y+h)
}

func (packer *Packer) split(image, space pixel.Rect) *createdSplits {
	w := space.W() - image.W()
	h := space.H() - image.H()

	if w < 0 || h < 0 {
		// failed
		return splitsFailed()
	} else if w == 0 && h == 0 {
		// perfectly fit case
		return &createdSplits{}
	} else if w > 0 && h == 0 {
		r := rect(space.Min.X+image.W(), space.Min.Y, w, image.H())
		return splits(r)
	} else if w == 0 && h > 0 {
		r := rect(space.Min.X, space.Min.Y+image.H(), image.W(), h)
		return splits(r)
	}

	var smaller, larger pixel.Rect
	if w > h {
		smaller = rect(space.Min.X+image.W(), space.Min.Y, w, image.H())
		larger = rect(space.Min.X, space.Min.Y+image.H(), image.W(), h)
	} else {
		smaller = rect(space.Min.X, space.Min.Y+image.H(), image.W(), h)
		larger = rect(space.Min.X+image.W(), space.Min.Y, w, space.H())
	}

	return splits(smaller, larger)
}

type createdSplits struct {
	count           int
	smaller, bigger pixel.Rect
}

func splitsFailed() *createdSplits {
	return &createdSplits{
		count: -1,
	}
}

func splits(rects ...pixel.Rect) (s *createdSplits) {
	s = &createdSplits{
		count:   len(rects),
		smaller: rects[0],
	}

	if s.count == 2 {
		s.bigger = rects[1]
	}

	return
}

type spaceSorter struct {
	spaces []pixel.Rect
}

func (sorter spaceSorter) Swap(i, j int) {
	sorter.spaces[i], sorter.spaces[j] = sorter.spaces[j], sorter.spaces[i]
}

func (sorter spaceSorter) Len() int {
	return len(sorter.spaces)
}

func (sorter spaceSorter) Less(i, j int) bool {
	return sorter.spaces[i].Area() < sorter.spaces[j].Area()
}
