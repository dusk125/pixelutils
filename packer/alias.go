package packer

import (
	"github.com/faiface/pixel"
)

type AliasPacker struct {
	base  *Packer
	alias map[interface{}]int
}

func NewAliasPacker(width, height int, flags uint8) (packer *AliasPacker) {
	packer = &AliasPacker{
		base:  NewPacker(width, height, flags),
		alias: make(map[interface{}]int),
	}

	return
}

// Gets the internal integer ID of the given alias
func (packer AliasPacker) IdOf(alias interface{}) int {
	return packer.alias[alias]
}

func (packer AliasPacker) AliasOf(id int) interface{} {
	for k, v := range packer.alias {
		if v == id {
			return k
		}
	}

	return nil
}

// Inserts the image with the given id into the texture space; default values.
func (packer *AliasPacker) Insert(alias interface{}, image *pixel.Sprite) (err error) {
	return packer.InsertV(alias, image, 0)
}

// Inserts the image with the given id and additional insertion flags.
func (packer *AliasPacker) InsertV(alias interface{}, image *pixel.Sprite, flags uint8) (err error) {
	id := packer.base.GenerateId()
	packer.alias[alias] = id
	return packer.base.InsertV(id, image, flags)
}

// Replaces replaces the given id with the new sprite (optimizing the atlas).
//	If the id doesn't exist in the atlas, this acts like a normal insert with optimization.
func (packer *AliasPacker) Replace(alias interface{}, sprite *pixel.Sprite) (err error) {
	if _, has := packer.alias[alias]; !has {
		return packer.Insert(alias, sprite)
	}
	return packer.base.Replace(packer.IdOf(alias), sprite)
}

// Returns the bounds of the given texture id
func (packer AliasPacker) BoundsOf(alias interface{}) pixel.Rect {
	return packer.base.images[packer.IdOf(alias)]
}

// Returns a copy of the sprite with the given ID
func (packer AliasPacker) SpriteFrom(alias interface{}) *pixel.Sprite {
	return packer.base.SpriteFrom(packer.IdOf(alias))
}

// Returns the center location of the packer's internal texture
func (packer AliasPacker) Center() pixel.Vec {
	return packer.base.Center()
}

// Returns the bounds of the packer's internal texture
func (packer AliasPacker) Bounds() pixel.Rect {
	return packer.base.Bounds()
}

// Generates and returns a picture data representation of the internal texture
func (packer *AliasPacker) Picture() pixel.Picture {
	return packer.base.Picture()
}
