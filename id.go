package pixelutils

import "sync"

// IDGen is a simple, thread safe, integer id generator
type IDGen struct {
	sync.Mutex
	nextID int
}

// Gen generates the next id and returns it
func (gen *IDGen) Gen() (id int) {
	gen.Lock()
	defer gen.Unlock()

	id = gen.nextID
	gen.nextID++

	return
}
