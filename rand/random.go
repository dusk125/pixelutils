package rand

import (
	"math/rand"
	"sort"
	"time"
)

type WeightedRandom struct {
	rand    *rand.Rand
	choices []RandChoice
}

type RandChoice struct {
	Chance float64
	Value  interface{}
}

func NewWeightedRandom(choices []RandChoice, seed int64) *WeightedRandom {
	if seed == 0 {
		seed = time.Hour.Nanoseconds()
	}

	sort.Slice(choices, func(i, j int) bool { return choices[i].Chance < choices[j].Chance })

	return &WeightedRandom{
		rand:    rand.New(rand.NewSource(seed)),
		choices: choices[:],
	}
}

func (wr WeightedRandom) Get() interface{} {
	var total float64
	for _, v := range wr.choices {
		total += v.Value.(float64)
	}

	rand := wr.rand.Float64() * total
	for _, v := range wr.choices {
		if v.Chance > rand {
			return v.Value
		}

		rand -= v.Chance
	}
	return nil
}
