package packer

import (
	"fmt"
	"log"
	"sort"

	"github.com/faiface/pixel/pixelgl"

	"golang.org/x/image/colornames"

	"github.com/dusk125/pixelutils"
	"github.com/faiface/pixel/imdraw"

	"github.com/faiface/pixel"
)

type Packer struct {
	bounds       pixel.Rect
	emptySpaces  spaceList
	filledSpaces spaceList
	pic          *pixel.PictureData
	flags        uint8
	im           *imdraw.IMDraw
	images       map[int]pixel.Rect
	id           *pixelutils.IDGen
	dirty        bool
	glpic        pixelgl.GLPicture

	// TODO remove; debug stuff
	sprite *pixel.Sprite
}

type spaceList []pixel.Rect

func (s spaceList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s spaceList) Len() int           { return len(s) }
func (s spaceList) Less(i, j int) bool { return s[i].Area() < s[j].Area() }

const (
	PackerAllowGrowth uint8 = 1 << iota
	PackerDebugDraw
	PackerDefragOnInsert
)

const (
	InsertFlipped uint8 = 1 << iota
)

func NewPacker(width, height int, flags uint8) *Packer {
	packer := &Packer{
		bounds: pixel.R(0, 0, float64(width), float64(height)),
		flags:  flags,
		id:     pixelutils.NewIDGen(),
	}

	packer.pic = pixel.MakePictureData(packer.bounds)

	if packer.hasFlag(PackerDebugDraw) {
		packer.im = imdraw.New(nil)
		packer.im.Color = colornames.Red
		packer.im.Push(packer.bounds.Min, packer.bounds.Max)
		packer.im.Rectangle(5)
	}

	packer.emptySpaces = make(spaceList, 1)
	packer.emptySpaces[0] = packer.bounds
	packer.filledSpaces = make(spaceList, 0)

	packer.images = make(map[int]pixel.Rect)

	return packer
}

func (packer *Packer) remove(i int) (removed pixel.Rect) {
	removed = packer.emptySpaces[i]
	packer.emptySpaces = append(packer.emptySpaces[:i], packer.emptySpaces[i+1:]...)
	return
}

func (packer Packer) hasFlag(flag uint8) bool {
	return packer.flags&flag != 0
}

func (packer *Packer) Defrag() {
	spaces := make(spaceList, len(packer.filledSpaces))
	copy(spaces, packer.filledSpaces)
	sort.Sort(sort.Reverse(spaces))

	packer.emptySpaces = packer.emptySpaces[:0]
	packer.filledSpaces = packer.filledSpaces[:0]
	packer.emptySpaces = append(packer.emptySpaces, packer.bounds)

	for _, space := range spaces {
		bounds := pixel.R(0, 0, space.W(), space.H())
		if _, _, err := packer.insert(bounds); err != nil {
			log.Fatalln(err)
		}
	}
}

func (packer *Packer) insert(bounds pixel.Rect) (space pixel.Rect, id int, err error) {
	candidateIndex := 0
	if len(packer.emptySpaces) > 1 {
		found := false
		for i, space := range packer.emptySpaces {
			if bounds.W() <= space.W() && bounds.H() <= space.H() {
				candidateIndex = i
				found = true
				break
			}
		}

		if !found {
			return pixel.ZR, -1, fmt.Errorf("Couldn't find an empty space")
		}
	}

	space = packer.remove(candidateIndex)
	packer.filledSpaces = append(packer.filledSpaces, space)

	s := packer.split(bounds, space)

	if s.count == -1 {
		return pixel.ZR, -1, fmt.Errorf("Failed to split the space")
	}

	if s.count == 2 {
		packer.emptySpaces = append(packer.emptySpaces, s.bigger)
	}
	packer.emptySpaces = append(packer.emptySpaces, s.smaller)

	sort.Sort(packer.emptySpaces)

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

	id = packer.id.Gen()
	packer.images[id] = rect(space.Min.X, space.Min.Y, bounds.W(), bounds.H())

	return space, id, nil
}

func (packer *Packer) Insert(image *pixel.Sprite) (id int, err error) {
	return packer.InsertV(image, 0)
}

func (packer *Packer) InsertV(image *pixel.Sprite, flags uint8) (id int, err error) {
	bounds := image.Picture().Bounds()
	pic := image.Picture().(*pixel.PictureData)

	var space pixel.Rect
	if space, id, err = packer.insert(bounds); err != nil {
		return -1, err
	}

	for y := 0; y < int(pic.Bounds().H()); y++ {
		for x := 0; x < int(pic.Bounds().W()); x++ {
			i := packer.pic.Index(pixel.V(space.Min.X+float64(x), space.Min.Y+float64(y)))
			var ii int
			if flags&InsertFlipped != 0 {
				ii = pic.Index(pixel.V(float64(x), (pic.Bounds().H()-1)-float64(y)))
			} else {
				ii = pic.Index(pixel.V(float64(x), float64(y)))
			}
			packer.pic.Pix[i] = pic.Pix[ii]
		}
	}

	packer.dirty = true

	return id, nil
}

func (packer Packer) BoundsOf(id int) pixel.Rect {
	return packer.images[id]
}

func (packer Packer) Center() pixel.Vec {
	return packer.Bounds().Center()
}

func (packer Packer) Bounds() pixel.Rect {
	return packer.bounds
}

func (packer *Packer) Picture() pixel.Picture {
	if packer.dirty {
		packer.glpic = pixelgl.NewGLPicture(packer.pic)
		packer.dirty = false
	}
	return packer.glpic
}

func (packer *Packer) Draw(t pixel.Target, matrix pixel.Matrix) {
	if packer.sprite == nil {
		packer.sprite = pixel.NewSprite(packer.Picture(), packer.Picture().Bounds())
	}

	packer.sprite.Draw(t, matrix)

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
