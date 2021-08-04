package packer

import (
	"errors"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"sort"

	"github.com/dusk125/pixelutils"
	"github.com/faiface/pixel"
)

// This texture packer algorithm is based on this project
// https://github.com/TeamHypersomnia/rectpack2D

type PackFlags uint8
type CreateFlags uint8

const (
	AllowGrowth CreateFlags = 1 << iota // Should the packer space try to grow larger to fit oversized images
)

const (
	InsertFlipped PackFlags = 1 << iota // Should the sprite be inserted into the packer upside-down
)

type Packer struct {
	bounds      pixel.Rect
	emptySpaces []pixel.Rect
	queued      []queuedData
	rects       map[int]pixel.Rect
	images      map[int]*pixel.PictureData
	flags       CreateFlags
	batch       *pixel.Batch
	pic         *pixel.PictureData
}

// Creates a new packer instance
func NewPacker(width, height int, flags CreateFlags) (pack *Packer) {
	bounds := pixel.R(0, 0, float64(width), float64(height))
	pack = &Packer{
		bounds:      bounds,
		flags:       flags,
		emptySpaces: []pixel.Rect{bounds},
		rects:       make(map[int]pixel.Rect),
		images:      make(map[int]*pixel.PictureData),
		queued:      make([]queuedData, 0),
	}
	return
}

// Helper to load a sprite from a file, and directly add it to the packer
func (pack *Packer) LoadAndInsert(id int, path string) (err error) {
	var (
		sprite *pixel.Sprite
	)
	if sprite, err = pixelutils.LoadSprite(path); err != nil {
		return
	}
	return pack.InsertPictureData(id, sprite.Picture().(*pixel.PictureData))
}

// Inserts PictureData into the packer
func (pack *Packer) InsertPictureData(id int, pic *pixel.PictureData) (err error) {
	pack.queued = append(pack.queued, queuedData{id: id, pic: pic})
	return
}

// Helper to find the smallest empty space that'll fit the given bounds
func (pack Packer) find(bounds pixel.Rect) (index int, found bool) {
	for i, space := range pack.emptySpaces {
		if bounds.W() <= space.W() && bounds.H() <= space.H() {
			return i, true
		}
	}
	return
}

// Helper to remove a canidate empty space and return it
func (pack *Packer) remove(i int) (removed pixel.Rect) {
	removed = pack.emptySpaces[i]
	pack.emptySpaces = append(pack.emptySpaces[:i], pack.emptySpaces[i+1:]...)
	return
}

// Helper to increase the size of the internal texture and readd the queued textures to keep it defragmented
func (pack *Packer) grow(growBy pixel.Vec, endex int) (err error) {
	pack.bounds = pack.bounds.Resized(pack.bounds.Min, pack.bounds.Max.Add(growBy))
	pack.emptySpaces = []pixel.Rect{pack.bounds}

	for _, data := range pack.queued[0:endex] {
		if err = pack.insert(data); err != nil {
			return
		}
	}

	return
}

// Helper to segment a found space so that the given data can fit in what's left
func (pack *Packer) insert(data queuedData) (err error) {
	var (
		s            *createdSplits
		bounds       = data.pic.Bounds()
		index, found = pack.find(bounds)
	)

	if !found {
		return ErrGrowthFailed
	}

	space := pack.remove(index)
	if s, err = split(bounds, space); err != nil {
		return
	}

	if s.hasBig {
		pack.emptySpaces = append(pack.emptySpaces, s.bigger)
	}
	if s.hasSmall {
		pack.emptySpaces = append(pack.emptySpaces, s.smaller)
	}

	sort.Slice(pack.emptySpaces, func(i, j int) bool {
		return pack.emptySpaces[i].Area() < pack.emptySpaces[j].Area()
	})

	pack.rects[data.id] = rect(space.Min.X, space.Min.Y, bounds.W(), bounds.H())
	pack.images[data.id] = data.pic
	return
}

// Pack takes the added textures and packs them into the packer texture, growing the texture if necessary.
func (pack *Packer) Pack(flags PackFlags) (err error) {
	sort.Slice(pack.queued, func(i, j int) bool {
		return pack.queued[i].pic.Bounds().Area() > pack.queued[j].pic.Bounds().Area()
	})

	for i, data := range pack.queued {
		var (
			bounds   = data.pic.Bounds()
			_, found = pack.find(bounds)
		)

		if !found {
			if pack.flags&AllowGrowth == 0 {
				return ErrorNoEmptySpace
			}

			if err = pack.grow(pixel.V(bounds.Size().X, bounds.Size().Y), i); err != nil {
				return
			}
		}

		if err = pack.insert(data); err != nil {
			return
		}
	}

	pack.pic = pixel.MakePictureData(pack.bounds)
	for id, pic := range pack.images {
		for x := 0; x < int(pic.Bounds().W()); x++ {
			for y := 0; y < int(pic.Bounds().H()); y++ {
				rect := pack.rects[id]
				dstI := pack.pic.Index(pixel.V(float64(x)+rect.Min.X, float64(y)+rect.Min.Y))
				var srcI int
				if flags&InsertFlipped != 0 {
					srcI = pic.Index(pixel.V(float64(x), (pic.Bounds().H()-1)-float64(y)))
				} else {
					srcI = pic.Index(pixel.V(float64(x), float64(y)))
				}

				pack.pic.Pix[dstI] = pic.Pix[srcI]
			}
		}
	}
	pack.batch = pixel.NewBatch(&pixel.TrianglesData{}, pack.pic)

	pack.queued = pack.queued[:0]
	pack.emptySpaces = pack.emptySpaces[:0]
	pack.images = nil

	return
}

// Saves the internal texture as a file on disk, the output type is defined by the filename extension
func (pack *Packer) Save(filename string) (err error) {
	var (
		file *os.File
	)

	if err = os.Remove(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}

	if file, err = os.Create(filename); err != nil {
		return
	}
	defer file.Close()

	img := pack.pic.Image()

	switch path.Ext(filename) {
	case ".png":
		err = png.Encode(file, img)
	case ".jpeg", ".jpg":
		err = jpeg.Encode(file, img, nil)
	default:
		err = ErrUnsupportedSaveExt
	}
	if err != nil {
		return
	}

	return
}

// Draws the given texture to the batch
func (pack *Packer) Draw(id int, m pixel.Matrix) {
	sprite := pixel.NewSprite(pack.pic, pack.rects[id])
	sprite.Draw(pack.batch, m)
}

// Draws the internal batch to the given target
func (pack *Packer) DrawTo(t pixel.Target) {
	pack.batch.Draw(t)
}

// Clear the internal batch of drawn sprites
func (pack *Packer) Clear() {
	pack.batch.Clear()
}
