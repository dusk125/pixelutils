package packer

import (
	"errors"
	"sort"

	"github.com/faiface/pixel"
)

type CreateFlags uint8
type InsertFlags uint8

var (
	ErrorNoEmptySpace       = errors.New("Couldn't find an empty space")
	ErrorNoSpaceAfterGrowth = errors.New("Failed to find space after growth")
	ErrorSplitFailed        = errors.New("Split failed")
)

// NewPacker-time flags
const (
	BaseAllowGrowth CreateFlags = 1 << iota // Should the packer space try to grow larger to fit oversized images
	BaseDebugDraw                           // Show the lines of the empty spaces when drawing the texture as an entire sprite
)

// Insert-time flags
const (
	BaseInsertFlipped InsertFlags = 1 << iota // Flip the sprite upside-down on insert
)

type Base struct {
	bounds      pixel.Rect
	emptySpaces spaceList
	pic         *pixel.PictureData
	flags       CreateFlags
	images      map[int]pixel.Rect
	dirty       bool
	batch       *pixel.Batch
}

func NewBase(width, height int, flags CreateFlags) (base *Base) {
	bounds := pixel.R(0, 0, float64(width), float64(height))
	base = &Base{
		bounds:      bounds,
		flags:       flags,
		pic:         pixel.MakePictureData(bounds),
		emptySpaces: spaceList{bounds},
		images:      make(map[int]pixel.Rect),
	}
	base.batch = pixel.NewBatch(&pixel.TrianglesData{}, base.pic)

	return
}

func (pack Base) find(bounds pixel.Rect) (int, bool) {
	for i, space := range pack.emptySpaces {
		if bounds.W() <= space.W() && bounds.H() <= space.H() {
			return i, true
		}
	}
	return -1, false
}

func (pack Base) spriteFrom(id int) *pixel.Sprite {
	image := pack.images[id]
	return pixel.NewSprite(pack.pic, image)
}

func (pack *Base) grow(size pixel.Vec) (err error) {
	sprites := pack.extract()
	pack.bounds = rect(0, 0, pack.bounds.W()+size.X, pack.bounds.H()+size.Y)
	return pack.fill(sprites)
}

func (pack *Base) remove(i int) (removed pixel.Rect) {
	removed = pack.emptySpaces[i]
	pack.emptySpaces = append(pack.emptySpaces[:i], pack.emptySpaces[i+1:]...)
	return
}

func (pack *Base) insert(bounds pixel.Rect, id int) (r pixel.Rect, err error) {
	var (
		s *createdSplits
	)
	candidateIndex, found := pack.find(bounds)

	if !found {
		if pack.flags&BaseAllowGrowth == 0 {
			return pixel.ZR, ErrorNoEmptySpace
		}

		if err = pack.grow(bounds.Size()); err != nil {
			return
		}

		candidateIndex, found = pack.find(bounds)
		if !found {
			return pixel.ZR, ErrorNoSpaceAfterGrowth
		}
	}

	space := pack.remove(candidateIndex)
	if s, err = split(bounds, space); err != nil {
		return pixel.ZR, err
	}

	if s.count == 2 {
		pack.emptySpaces = append(pack.emptySpaces, s.bigger)
	}
	pack.emptySpaces = append(pack.emptySpaces, s.smaller)

	sort.Sort(pack.emptySpaces)

	pack.images[id] = rect(space.Min.X, space.Min.Y, bounds.W(), bounds.H())

	return pack.images[id], nil
}

func (pack *Base) makeDirty() {
	pack.dirty = true
}

func (pack *Base) Insert(id int, image *pixel.Sprite) error {
	return pack.InsertV(id, image, 0)
}

func (pack *Base) InsertV(id int, image *pixel.Sprite, flags InsertFlags) error {
	pic := image.Picture().(*pixel.PictureData)

	return pack.InsertPictureDataV(id, pic, flags)
}

func (pack *Base) InsertPictureData(id int, pic *pixel.PictureData) error {
	return pack.InsertPictureDataV(id, pic, 0)
}

func (pack *Base) InsertPictureDataV(id int, pic *pixel.PictureData, flags InsertFlags) (err error) {
	var (
		bounds = pic.Bounds()
		space  pixel.Rect
	)

	if space, err = pack.insert(bounds, id); err != nil {
		return
	}

	for y := 0; y < int(bounds.H()); y++ {
		for x := 0; x < int(bounds.W()); x++ {
			i := pack.pic.Index(pixel.V(space.Min.X+float64(x), space.Min.Y+float64(y)))
			var ii int
			if flags&BaseInsertFlipped != 0 {
				ii = pic.Index(pixel.V(float64(x), (bounds.H()-1)-float64(y)))
			} else {
				ii = pic.Index(pixel.V(float64(x), float64(y)))
			}
			pack.pic.Pix[i] = pic.Pix[ii]
		}
	}

	pack.makeDirty()

	return
}

func (pack Base) extract() (sprites spriteList) {
	sprites = make(spriteList, 0)
	for id := range pack.images {
		sprite := pack.spriteFrom(id)
		sprites = append(sprites, idSprite{id: id, sprite: sprite})
	}
	sort.Sort(sprites)
	return
}

func (pack *Base) fill(sprites spriteList) (err error) {
	pack.emptySpaces = pack.emptySpaces[:0]
	pack.emptySpaces = append(pack.emptySpaces, pack.bounds)
	pack.pic = pixel.MakePictureData(pack.bounds)
	pack.batch = pixel.NewBatch(&pixel.TrianglesData{}, pack.pic)
	pack.makeDirty()

	for _, cont := range sprites {
		if err = pack.Insert(cont.id, cont.sprite); err != nil {
			return
		}
	}
	return
}

func (pack *Base) Optimize() (err error) {
	return pack.fill(pack.extract())
}

func (pack *Base) Draw(id int, m pixel.Matrix) {
	sprite := pack.spriteFrom(id)

	sprite.Draw(pack.batch, m)
}

func (pack *Base) DrawTo(target pixel.Target) {
	pack.batch.Draw(target)
}

func (pack *Base) Clear() {
	pack.batch.Clear()
}
