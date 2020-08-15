package pixelutils

import (
	"github.com/faiface/pixel"
	"github.com/pzsz/voronoi"
)

// This is an pixel-conversion interface for https://github.com/pzsz/voronoi

type VoronoiCell struct {
	ID        int
	Site      pixel.Vec
	Halfedges []pixel.Line
}

type Voronoi struct {
	siteMap map[int]voronoi.Vertex
	sites   []voronoi.Vertex
}

// helper to convert voronoi's vertex to pixel.Vec
func convertVVertex(vertex voronoi.Vertex) pixel.Vec {
	return pixel.V(vertex.X, vertex.Y)
}

// Creates and returns a new pixel-compatible Voronoi
func NewVoronoi() (v *Voronoi) {
	v = &Voronoi{}

	v.siteMap = make(map[int]voronoi.Vertex)
	v.sites = make([]voronoi.Vertex, 0)

	return
}

// Inserts the vector into the voronoi list with the given identifier
func (v *Voronoi) Insert(id int, pos pixel.Vec) {
	vert := voronoi.Vertex{
		X: pos.X,
		Y: pos.Y,
	}
	v.sites = append(v.sites, vert)
	v.siteMap[id] = vert
}

// Computes the voronoi diagram, constrained by the given bounding box, from the nodes inserted into the list
//	If closeCells == true, edges from bounding box will be included in the diagram.
func (v *Voronoi) Compute(boundingBox pixel.Rect, closeCells bool) map[int]VoronoiCell {
	bb := voronoi.NewBBox(boundingBox.Min.X, boundingBox.Max.X, boundingBox.Min.Y, boundingBox.Max.Y)
	diagram := voronoi.ComputeDiagram(v.sites, bb, closeCells)
	return v.convert(diagram)
}

// helper function to conver the internal voronoi diagram to a pixel-compatible one
func (v *Voronoi) convert(diagram *voronoi.Diagram) map[int]VoronoiCell {
	cells := make(map[int]VoronoiCell)

	for _, vCell := range diagram.Cells {
		ID := -1
		for id, v := range v.siteMap {
			if v.X == vCell.Site.X && v.Y == vCell.Site.Y {
				ID = id
				break
			}
		}

		cell := VoronoiCell{
			ID:   ID,
			Site: convertVVertex(vCell.Site),
		}
		cell.Halfedges = make([]pixel.Line, 0)

		for _, edge := range vCell.Halfedges {
			cell.Halfedges = append(cell.Halfedges, pixel.L(convertVVertex(edge.GetStartpoint()), convertVVertex(edge.GetEndpoint())))
		}

		cells[ID] = cell
	}

	return cells
}
