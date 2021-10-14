package packer

import (
	"image"

	"github.com/dusk125/rectpack"
	"github.com/faiface/pixel"
)

func imageRectToPixelRect(r image.Rectangle) (pr pixel.Rect) {
	return pixel.R(float64(r.Min.X), float64(r.Min.Y), float64(r.Max.X), float64(r.Max.Y))
}

type Packer struct {
	rectpack.Packer
	pic   pixel.Picture
	batch *pixel.Batch
}

func New() (pack *Packer) {
	pack = &Packer{
		Packer: *rectpack.NewPacker(rectpack.PackerCfg{}),
	}
	return
}

func (pack *Packer) Pack() (err error) {
	if err = pack.Packer.Pack(); err != nil {
		return
	}

	pack.pic = pixel.PictureDataFromImage(pack.Packer.Image())
	pack.batch = pixel.NewBatch(&pixel.TrianglesData{}, pack.pic)

	return
}

func (pack *Packer) BoundsOf(id int) pixel.Rect {
	return imageRectToPixelRect(pack.Get(id))
}

func (pack *Packer) Draw(t pixel.Target) {
	pack.batch.Draw(t)
}

func (pack *Packer) DrawSub(id int, m pixel.Matrix) {
	r := imageRectToPixelRect(pack.Get(id))
	s := pixel.NewSprite(pack.pic, r)
	s.Draw(pack.batch, m)
}

func (pack *Packer) Picture() pixel.Picture {
	return pack.pic
}

func (pack *Packer) Bounds() pixel.Rect {
	return pack.pic.Bounds()
}
