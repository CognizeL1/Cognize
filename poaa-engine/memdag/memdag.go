package memdag

import (
	"sync"

	"github.com/cognize/axon/poaa-engine/dag"
	sdkmath "cosmossdk.io/math"
	"github.com/hashicorp/golang-lru/v2"
)

type MemDAG struct {
	vertices map[[32]byte]*dag.Vertex
	tips     *dag.TipsPool
	cache    *lru.Cache[[32]byte, *dag.Vertex]
	mu       sync.RWMutex
	depth    uint64
}

func NewMemDAG() (*MemDAG, error) {
	cache, err := lru.New[[32]byte, *dag.Vertex](5000)
	if err != nil {
		return nil, err
	}
	return &MemDAG{
		vertices: make(map[[32]byte]*dag.Vertex),
		tips:     dag.NewTipsPool(),
		cache:    cache,
	}, nil
}

func (m *MemDAG) AddVertex(v *dag.Vertex) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if v.Hash == [32]byte{} {
		v.Hash = v.ComputeHash()
	}

	m.vertices[v.Hash] = v
	m.cache.Add(v.Hash, v)

	for _, p := range v.Parents {
		if p != [32]byte{} {
			m.tips.Remove(p)
		}
	}
	m.tips.Add(v)

	if v.Depth > m.depth {
		m.depth = v.Depth
	}

	return nil
}

func (m *MemDAG) GetVertex(hash [32]byte) (*dag.Vertex, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if v, ok := m.cache.Get(hash); ok {
		return v, true
	}

	v, ok := m.vertices[hash]
	if ok {
		m.cache.Add(hash, v)
	}
	return v, ok
}

func (m *MemDAG) HasVertex(hash [32]byte) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.vertices[hash]
	return ok
}

func (m *MemDAG) GetTips() []*dag.Vertex {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.tips.GetAll()
}

func (m *MemDAG) GetChildren(parentHash [32]byte) [][32]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var children [][32]byte
	for h, v := range m.vertices {
		for _, p := range v.Parents {
			if p == parentHash {
				var childHash [32]byte
				copy(childHash[:], h[:])
				children = append(children, childHash)
			}
		}
	}
	return children
}

func (m *MemDAG) GetConfirmedVertices() []*dag.Vertex {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var confirmed []*dag.Vertex
	for _, v := range m.vertices {
		if v.Confirmed {
			confirmed = append(confirmed, v)
		}
	}
	return confirmed
}

func (m *MemDAG) GetVerticesByLayer(layer dag.ConfirmationLayer) []*dag.Vertex {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*dag.Vertex
	for _, v := range m.vertices {
		if v.Layer == layer {
			result = append(result, v)
		}
	}
	return result
}

func (m *MemDAG) ConfirmVertex(hash [32]byte, confirmRecord dag.ConfirmRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok := m.vertices[hash]
	if !ok {
		return dag.ErrVertexNotFound
	}

	v.Confirmed = true
	v.Confirmers = append(v.Confirmers, confirmRecord)
	v.TotalWeight = v.TotalWeight.Add(confirmRecord.Weight)
	v.Layer = dag.LayerSoft
	return nil
}

func (m *MemDAG) PromoteVertex(hash [32]byte, layer dag.ConfirmationLayer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok := m.vertices[hash]
	if !ok {
		return dag.ErrVertexNotFound
	}

	v.Layer = layer
	if layer >= dag.LayerFast {
		v.FinalityAt = dag.GetCurrentTimestamp()
	}
	return nil
}

func (m *MemDAG) GetDepth() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.depth
}

func (m *MemDAG) SetDepth(depth uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.depth = depth
}

func (m *MemDAG) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.vertices)
}

func (m *MemDAG) GetAllVertices() []*dag.Vertex {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*dag.Vertex, 0, len(m.vertices))
	for _, v := range m.vertices {
		result = append(result, v)
	}
	return result
}

func (m *MemDAG) PruneBelow(depth uint64) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	pruned := 0
	for h, v := range m.vertices {
		if v.Depth < depth {
			delete(m.vertices, h)
			m.cache.Remove(h)
			pruned++
		}
	}
	return pruned
}

func (m *MemDAG) CalculateWeight(hash [32]byte) sdkmath.LegacyDec {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.vertices[hash]
	if !ok {
		return sdkmath.LegacyZeroDec()
	}

	totalWeight := sdkmath.LegacyZeroDec()
	for _, c := range v.Confirmers {
		totalWeight = totalWeight.Add(c.Weight)
	}
	return totalWeight
}

func (m *MemDAG) GetAncestors(hash [32]byte, maxDepth uint64) []*dag.Vertex {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.vertices[hash]
	if !ok {
		return nil
	}

	var ancestors []*dag.Vertex
	visited := make(map[[32]byte]bool)

	queue := [][32]byte{v.Parents[0], v.Parents[1]}
	for len(queue) > 0 && uint64(len(ancestors)) < maxDepth {
		current := queue[0]
		queue = queue[1:]

		if current == [32]byte{} || visited[current] {
			continue
		}
		visited[current] = true

		anc, ok := m.vertices[current]
		if !ok {
			continue
		}

		ancestors = append(ancestors, anc)
		queue = append(queue, anc.Parents[0], anc.Parents[1])
	}

	return ancestors
}

func (m *MemDAG) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.vertices = make(map[[32]byte]*dag.Vertex)
	m.tips = dag.NewTipsPool()
	m.cache.Purge()
	m.depth = 0
}
