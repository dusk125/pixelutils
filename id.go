package pixelutils

import "sync"

type IDGen struct {
	nextID int
	idLock sync.Mutex
}

func NewIDGen() *IDGen {
	return &IDGen{}
}

func (gen *IDGen) Gen() int {
	gen.idLock.Lock()
	defer gen.idLock.Unlock()

	id := gen.nextID
	gen.nextID++

	return id
}
