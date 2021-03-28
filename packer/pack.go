package packer

import (
	"fmt"
	"log"
	"sort"

	"github.com/dusk125/pixelutils"

	"github.com/faiface/pixel/pixelgl"

	"golang.org/x/image/colornames"

	"github.com/faiface/pixel/imdraw"

	"github.com/faiface/pixel"
)

// This texture packer algorithm is based on this project
// https://github.com/TeamHypersomnia/rectpack2D

type Packer struct {
	bounds      pixel.Rect
	emptySpaces spaceList
	pic         *pixel.PictureData
	flags       uint8
	im          *imdraw.IMDraw
	images      map[int]pixel.Rect
	dirty       bool
	glpic       pixelgl.GLPicture
	sprite      *pixel.Sprite
	id          *pixelutils.IDGen
}

// NewPacker-time flags
const (
	AllowGrowth uint8 = 1 << iota // Should the packer space try to grow larger to fit oversized images
	DebugDraw                     // Show the lines of the empty spaces when drawing the texture as an entire sprite
)

// Insert-time flags
const (
	InsertFlipped    uint8 = 1 << iota // Flip the sprite upside-down on insert
	OptimizeOnInsert                   // When a new image is inserted, defragment the texture space
)

// NewPacker creates and returns a new texture packer
func NewPacker(width, height int, flags uint8) *Packer {
	bounds := pixel.R(0, 0, float64(width), float64(height))
	packer := &Packer{
		bounds:      bounds,
		flags:       flags,
		id:          pixelutils.NewIDGen(),
		pic:         pixel.MakePictureData(bounds),
		emptySpaces: make(spaceList, 1),
		images:      make(map[int]pixel.Rect),
	}

	if packer.hasFlag(DebugDraw) {
		packer.im = imdraw.New(nil)
		packer.im.Color = colornames.Red
		packer.im.Push(bounds.Min, bounds.Max)
		packer.im.Rectangle(5)
	}

	packer.emptySpaces[0] = bounds

	return packer
}

// remove removes the given image
func (packer *Packer) remove(i int) (removed pixel.Rect) {
	removed = packer.emptySpaces[i]
	packer.emptySpaces = append(packer.emptySpaces[:i], packer.emptySpaces[i+1:]...)
	return
}

// hasFlag is a helper to test bit flags
func (packer Packer) hasFlag(flag uint8) bool {
	return packer.flags&flag != 0
}

// Returns a copy of the sprite with the given ID
func (packer Packer) SpriteFrom(id int) *pixel.Sprite {
	subImage := packer.images[id]
	image := subImage.Moved(subImage.Min.Scaled(-1))
	pic := pixel.MakePictureData(image)

	for y := 0; y < int(image.H()); y++ {
		for x := 0; x < int(image.W()); x++ {
			v := pixel.V(float64(x), float64(y))
			pic.Pix[pic.Index(v)] = packer.pic.Pix[packer.pic.Index(v.Add(subImage.Min))]
		}
	}

	return pixel.NewSprite(pic, pic.Bounds())
}

// Grows the packer space by the given amount
func (packer *Packer) grow(growSize pixel.Vec) {
	packer.bounds = rect(0, 0, packer.bounds.W()+growSize.X, packer.bounds.H()+growSize.Y)
	packer.reinsert(false)
}

// Aids in optimizing and growing the texture space
func (packer *Packer) reinsert(optimze bool) {
	// Pull out of the sprites from the atlas
	sprites := make(spriteList, 0)
	for id := range packer.images {
		sprite := packer.SpriteFrom(id)
		sprites = append(sprites, idSprite{id: id, sprite: sprite})
	}

	if optimze {
		// Sort them largest to smallest
		sort.Sort(sort.Reverse(sprites))
	}

	// Clear out atlas and metadata
	packer.emptySpaces = packer.emptySpaces[:0]
	packer.emptySpaces = append(packer.emptySpaces, packer.bounds)
	packer.pic = pixel.MakePictureData(packer.bounds)
	packer.makeDirty()

	// Re-insert sprites
	for _, cont := range sprites {
		if err := packer.Insert(cont.id, cont.sprite); err != nil {
			log.Fatalln(err)
		}
	}
}

// Defragments the texture space to try and minimize wasted space
func (packer *Packer) Optimize() {
	packer.reinsert(true)
}

// Replaces replaces the given id with the new sprite (optimizing the atlas).
//	If the id doesn't exist in the atlas, this acts like a normal insert with optimization.
func (packer *Packer) Replace(id int, sprite *pixel.Sprite) (err error) {
	delete(packer.images, id)

	return packer.InsertV(id, sprite, OptimizeOnInsert)
}

// Looks for a space that can hold the given image bounds
func (packer *Packer) find(bounds pixel.Rect) (candidateIndex int, found bool) {
	for i, space := range packer.emptySpaces {
		if bounds.W() <= space.W() && bounds.H() <= space.H() {
			candidateIndex = i
			found = true
			return
		}
	}

	return -1, false
}

// Helper for actually inserting and splitting the image bounds
func (packer *Packer) insert(bounds pixel.Rect, id int) (space pixel.Rect, err error) {
	candidateIndex, found := packer.find(bounds)

	if !found {
		if !packer.hasFlag(AllowGrowth) {
			return pixel.ZR, fmt.Errorf("Couldn't find an empty space")
		}

		packer.grow(bounds.Size())
		candidateIndex, found = packer.find(bounds)
		if !found {
			return pixel.ZR, fmt.Errorf("Failed to find space after growth")
		}
	}

	space = packer.remove(candidateIndex)

	s := packer.split(bounds, space)

	if s.count == -1 {
		return pixel.ZR, fmt.Errorf("Failed to split the space")
	}

	if s.count == 2 {
		packer.emptySpaces = append(packer.emptySpaces, s.bigger)
	}
	packer.emptySpaces = append(packer.emptySpaces, s.smaller)

	sort.Sort(packer.emptySpaces)

	if packer.hasFlag(DebugDraw) {
		packer.im.Reset()
		packer.im.Clear()

		packer.im.Color = colornames.Red
		for _, item := range packer.emptySpaces {
			packer.im.Push(item.Min, item.Max)
			packer.im.Rectangle(5)
		}
	}

	packer.images[id] = rect(space.Min.X, space.Min.Y, bounds.W(), bounds.H())

	return space, nil
}

// External helper that generates a unique (to this packer instance) texture ID
func (packer *Packer) GenerateId() int {
	return packer.id.Gen()
}

// Inserts the image with the given id into the texture space; default values.
func (packer *Packer) Insert(id int, image *pixel.Sprite) (err error) {
	return packer.InsertV(id, image, 0)
}

// Inserts the image with the given id and additional insertion flags.
func (packer *Packer) InsertV(id int, image *pixel.Sprite, flags uint8) (err error) {
	pic := image.Picture().(*pixel.PictureData)

	return packer.InsertPictureDataV(id, pic, flags)
}

// Inserts the PictureData with the given id into the texture space; default values.
func (packer *Packer) InsertPictureData(id int, pic *pixel.PictureData) (err error) {
	return packer.InsertPictureDataV(id, pic, 0)
}

// Inserts the picturedata with the given id and additional insertion flags.
func (packer *Packer) InsertPictureDataV(id int, pic *pixel.PictureData, flags uint8) (err error) {
	bounds := pic.Bounds()

	if flags&OptimizeOnInsert != 0 {
		packer.Optimize()
	}

	var space pixel.Rect
	if space, err = packer.insert(bounds, id); err != nil {
		return err
	}

	for y := 0; y < int(bounds.H()); y++ {
		for x := 0; x < int(bounds.W()); x++ {
			i := packer.pic.Index(pixel.V(space.Min.X+float64(x), space.Min.Y+float64(y)))
			var ii int
			if flags&InsertFlipped != 0 {
				ii = pic.Index(pixel.V(float64(x), (bounds.H()-1)-float64(y)))
			} else {
				ii = pic.Index(pixel.V(float64(x), float64(y)))
			}
			packer.pic.Pix[i] = pic.Pix[ii]
		}
	}

	packer.makeDirty()

	return nil
}

// Helper to invalidate the internal texture
func (packer *Packer) makeDirty() {
	packer.dirty = true
	packer.sprite = nil
}

// Returns the bounds of the given texture id
func (packer Packer) BoundsOf(id int) pixel.Rect {
	return packer.images[id]
}

// Returns the center location of the packer's internal texture
func (packer Packer) Center() pixel.Vec {
	return packer.Bounds().Center()
}

// Returns the bounds of the packer's internal texture
func (packer Packer) Bounds() pixel.Rect {
	return packer.bounds
}

// Generates and returns a picture data representation of the internal texture
func (packer *Packer) Picture() pixel.Picture {
	if packer.dirty {
		packer.glpic = pixelgl.NewGLPicture(packer.pic)
		packer.dirty = false
	}
	return packer.glpic
}

// Draws the internal texture as a sprite; recommended for debug only
func (packer *Packer) Draw(t pixel.Target, matrix pixel.Matrix) {
	if packer.sprite == nil {
		packer.sprite = pixel.NewSprite(packer.Picture(), packer.Picture().Bounds())
	}

	packer.sprite.Draw(t, matrix)

	if packer.hasFlag(DebugDraw) {
		packer.im.Draw(t)
	}
}

// Helper to create a rectangle with x,y,w,h instead of x1,y1,x2,y2
func rect(x, y, w, h float64) pixel.Rect {
	return pixel.R(x, y, x+w, y+h)
}

// Returns the post-insert spaces of leftover space; if any
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
		smaller = rect(space.Min.X, space.Min.Y+image.H(), image.W(), h)
		larger = rect(space.Min.X+image.W(), space.Min.Y, w, space.H())
	} else {
		smaller = rect(space.Min.X+image.W(), space.Min.Y, w, image.H())
		larger = rect(space.Min.X, space.Min.Y+image.H(), space.W(), h)
	}

	return splits(smaller, larger)
}
