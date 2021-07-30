package packer

import (
	"sort"

	"github.com/dusk125/goutil/logger"
	"github.com/faiface/pixel"
)

type PackFlags uint8

const (
	PackInsertFlipped PackFlags = 1 << iota
)

type New struct {
	bounds      pixel.Rect
	emptySpaces spaceList
	queued      picDataList
	images      map[int]pixel.Rect
	flags       CreateFlags
	batch       *pixel.Batch
}

func NewNew(width, height int, flags CreateFlags) (pack *New) {
	bounds := pixel.R(0, 0, float64(width), float64(height))
	pack = &New{
		bounds:      bounds,
		flags:       flags,
		emptySpaces: spaceList{bounds},
		images:      make(map[int]pixel.Rect),
		queued:      make(picDataList, 0),
	}
	return
}

func (pack *New) InsertPictureData(id int, pic *pixel.PictureData) (err error) {
	pack.queued = append(pack.queued, struct {
		id  int
		pic *pixel.PictureData
	}{id: id, pic: pic})
	return
}

func (pack New) find(bounds pixel.Rect) (index int, found bool) {
	for i, space := range pack.emptySpaces {
		if bounds.W() <= space.W() && bounds.H() <= space.H() {
			return i, true
		}
	}
	return
}

func (pack *New) remove(i int) (removed pixel.Rect) {
	removed = pack.emptySpaces[i]
	pack.emptySpaces = append(pack.emptySpaces[:i], pack.emptySpaces[i+1:]...)
	return
}

func (pack *New) Pack() (err error) {
	sort.Slice(pack.queued, func(i, j int) bool {
		return pack.queued[i].pic.Bounds().Area() > pack.queued[j].pic.Bounds().Area()
	})

	for _, data := range pack.queued {
		var (
			s            *createdSplits
			bounds       = data.pic.Bounds()
			index, found = pack.find(bounds)
		)

		if !found {
			if pack.flags&BaseAllowGrowth == 0 {
				return ErrorNoEmptySpace
			}

			pack.bounds = rect(0, 0, pack.bounds.W()+bounds.Size().X, pack.bounds.H()+bounds.Size().Y)
			index, found = pack.find(bounds)
			if !found {
				return ErrorNoSpaceAfterGrowth
			}
		}

		space := pack.remove(index)
		if s, err = split(bounds, space); err != nil {
			return err
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

		pack.images[data.id] = rect(space.Min.X, space.Min.Y, bounds.W(), bounds.H())
	}

	logger.Debug("Final bounds:", pack.bounds)

	return
}
