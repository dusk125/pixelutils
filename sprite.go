package pixelutils

import (
	"image"
	"os"

	"github.com/faiface/pixel"
)

// Helper to load a sprite from file and make a Pixel.Sprite from it
func LoadSprite(path string) (sprite *pixel.Sprite, err error) {
	var data *pixel.PictureData
	if data, err = LoadPictureData(path); err != nil {
		return nil, err
	}

	return pixel.NewSprite(data, data.Bounds()), nil
}

// Helper to load a sprite from a file and make Pixel.PictureData from it
func LoadPictureData(path string) (data *pixel.PictureData, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return pixel.PictureDataFromImage(img), nil
}
