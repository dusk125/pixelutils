package pixelutils

import "sync"

// IDGen is a simple, thread safe, integer id generator
type IDGen struct {
	nextID int
	idLock sync.Mutex
}

func NewIDGen() *IDGen {
	return &IDGen{
		nextID: 0,
	}
}

// Gen generates the next id and returns it
func (gen *IDGen) Gen() (id int) {
	gen.idLock.Lock()
	defer gen.idLock.Unlock()

	id = gen.nextID
	gen.nextID++

	return
}
