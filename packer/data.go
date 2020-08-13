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

// container for the leftover space after split
type createdSplits struct {
	count           int
	smaller, bigger pixel.Rect
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
		count:   len(rects),
		smaller: rects[0],
	}

	if s.count == 2 {
		s.bigger = rects[1]
	}

	return
}
