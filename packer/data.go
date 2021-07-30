package packer

import "github.com/faiface/pixel"

// sortable list of rectangles
type spaceList []pixel.Rect

func (s spaceList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s spaceList) Len() int           { return len(s) }
func (s spaceList) Less(i, j int) bool { return s[i].Area() < s[j].Area() }

// container for holding sprite and id together to be put into a list
type idSprite struct {
	id     int
	sprite *pixel.Sprite
}

// sortable list of idSprites
type spriteList []idSprite

func (s spriteList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s spriteList) Len() int      { return len(s) }
func (s spriteList) Less(i, j int) bool {
	return s[i].sprite.Picture().Bounds().Area() < s[j].sprite.Picture().Bounds().Area()
}

type picDataList []struct {
	id  int
	pic *pixel.PictureData
}

// container for the leftover space after split
type createdSplits struct {
	hasSmall, hasBig bool
	count            int
	smaller, bigger  pixel.Rect
}

// helper if the split was invalid
func splitsFailed() *createdSplits {
	return &createdSplits{
		count: -1,
	}
}

// adds the given leftover spaces to this container
func splits(rects ...pixel.Rect) (s *createdSplits) {
	s = &createdSplits{
		count:    len(rects),
		hasSmall: true,
		smaller:  rects[0],
	}

	if s.count == 2 {
		s.hasBig = true
		s.bigger = rects[1]
	}

	return
}

func split(image, space pixel.Rect) (s *createdSplits, err error) {
	w := space.W() - image.W()
	h := space.H() - image.H()

	if w < 0 || h < 0 {
		return nil, ErrorSplitFailed
	} else if w == 0 && h == 0 {
		// perfectly fit case
		return &createdSplits{}, nil
	} else if w > 0 && h == 0 {
		r := rect(space.Min.X+image.W(), space.Min.Y, w, image.H())
		return splits(r), nil
	} else if w == 0 && h > 0 {
		r := rect(space.Min.X, space.Min.Y+image.H(), image.W(), h)
		return splits(r), nil
	}

	var smaller, larger pixel.Rect
	if w > h {
		smaller = rect(space.Min.X, space.Min.Y+image.H(), image.W(), h)
		larger = rect(space.Min.X+image.W(), space.Min.Y, w, space.H())
	} else {
		smaller = rect(space.Min.X+image.W(), space.Min.Y, w, image.H())
		larger = rect(space.Min.X, space.Min.Y+image.H(), space.W(), h)
	}

	return splits(smaller, larger), nil
}
