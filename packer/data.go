package packer

import (
	"errors"

	"github.com/faiface/pixel"
)

var (
	ErrorNoEmptySpace     = errors.New("Couldn't find an empty space")
	ErrorSplitFailed      = errors.New("Split failed")
	ErrGrowthFailed       = errors.New("A previously added texture failed to be added after packer growth")
	ErrUnsupportedSaveExt = errors.New("Unsupported save filename extension")
)

type queuedData struct {
	id  int
	pic *pixel.PictureData
}

// container for the leftover space after split
type createdSplits struct {
	hasSmall, hasBig bool
	count            int
	smaller, bigger  pixel.Rect
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

// helper function to create rectangles
func rect(x, y, w, h float64) pixel.Rect {
	return pixel.R(x, y, x+w, y+h)
}

// helper to split existing space
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
