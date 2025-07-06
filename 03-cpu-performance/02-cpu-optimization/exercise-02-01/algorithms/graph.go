package algorithms

import (
	"container/heap"
	"math"
	"math/rand"
)

// Graph represents a weighted directed graph
type Graph struct {
	Vertices int
	Edges    [][]Edge
}

// Edge represents a weighted edge
type Edge struct {
	To     int
	Weight int
}

// NewGraph creates a new graph with the specified number of vertices
func NewGraph(vertices int) *Graph {
	return &Graph{
		Vertices: vertices,
		Edges:    make([][]Edge, vertices),
	}
}

// AddEdge adds a weighted edge to the graph
func (g *Graph) AddEdge(from, to, weight int) {
	g.Edges[from] = append(g.Edges[from], Edge{To: to, Weight: weight})
}

// GenerateRandomGraph creates a random graph for testing
func GenerateRandomGraph(vertices, edges int) *Graph {
	g := NewGraph(vertices)
	
	for i := 0; i < edges; i++ {
		from := rand.Intn(vertices)
		to := rand.Intn(vertices)
		weight := rand.Intn(100) + 1 // Weight between 1 and 100
		g.AddEdge(from, to, weight)
	}
	
	return g
}

// DepthFirstSearch performs DFS traversal
func DepthFirstSearch(g *Graph, start int) []int {
	visited := make([]bool, g.Vertices)
	result := make([]int, 0)
	
	dfsHelper(g, start, visited, &result)
	return result
}

func dfsHelper(g *Graph, vertex int, visited []bool, result *[]int) {
	visited[vertex] = true
	*result = append(*result, vertex)
	
	for _, edge := range g.Edges[vertex] {
		if !visited[edge.To] {
			dfsHelper(g, edge.To, visited, result)
		}
	}
}

// BreadthFirstSearch performs BFS traversal
func BreadthFirstSearch(g *Graph, start int) []int {
	visited := make([]bool, g.Vertices)
	queue := []int{start}
	result := make([]int, 0)
	
	visited[start] = true
	
	for len(queue) > 0 {
		vertex := queue[0]
		queue = queue[1:]
		result = append(result, vertex)
		
		for _, edge := range g.Edges[vertex] {
			if !visited[edge.To] {
				visited[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
	}
	
	return result
}

// PriorityQueueItem represents an item in the priority queue
type PriorityQueueItem struct {
	Vertex   int
	Distance int
	Index    int
}

// PriorityQueue implements a min-heap for Dijkstra's algorithm
type PriorityQueue []*PriorityQueueItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Distance < pq[j].Distance
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityQueueItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

// Dijkstra implements Dijkstra's shortest path algorithm
func Dijkstra(g *Graph, start int) []int {
	distances := make([]int, g.Vertices)
	for i := range distances {
		distances[i] = math.MaxInt32
	}
	distances[start] = 0
	
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)
	heap.Push(&pq, &PriorityQueueItem{Vertex: start, Distance: 0})
	
	for pq.Len() > 0 {
		current := heap.Pop(&pq).(*PriorityQueueItem)
		
		if current.Distance > distances[current.Vertex] {
			continue
		}
		
		for _, edge := range g.Edges[current.Vertex] {
			newDistance := distances[current.Vertex] + edge.Weight
			
			if newDistance < distances[edge.To] {
				distances[edge.To] = newDistance
				heap.Push(&pq, &PriorityQueueItem{Vertex: edge.To, Distance: newDistance})
			}
		}
	}
	
	return distances
}

// FloydWarshall implements Floyd-Warshall all-pairs shortest path algorithm
func FloydWarshall(g *Graph) [][]int {
	n := g.Vertices
	dist := make([][]int, n)
	
	// Initialize distance matrix
	for i := 0; i < n; i++ {
		dist[i] = make([]int, n)
		for j := 0; j < n; j++ {
			if i == j {
				dist[i][j] = 0
			} else {
				dist[i][j] = math.MaxInt32
			}
		}
	}
	
	// Fill in direct edges
	for i := 0; i < n; i++ {
		for _, edge := range g.Edges[i] {
			dist[i][edge.To] = edge.Weight
		}
	}
	
	// Floyd-Warshall algorithm
	for k := 0; k < n; k++ {
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if dist[i][k] != math.MaxInt32 && dist[k][j] != math.MaxInt32 {
					if dist[i][k]+dist[k][j] < dist[i][j] {
						dist[i][j] = dist[i][k] + dist[k][j]
					}
				}
			}
		}
	}
	
	return dist
}

// BellmanFord implements Bellman-Ford algorithm for shortest paths with negative weights
func BellmanFord(g *Graph, start int) ([]int, bool) {
	distances := make([]int, g.Vertices)
	for i := range distances {
		distances[i] = math.MaxInt32
	}
	distances[start] = 0
	
	// Relax edges V-1 times
	for i := 0; i < g.Vertices-1; i++ {
		for u := 0; u < g.Vertices; u++ {
			if distances[u] != math.MaxInt32 {
				for _, edge := range g.Edges[u] {
					if distances[u]+edge.Weight < distances[edge.To] {
						distances[edge.To] = distances[u] + edge.Weight
					}
				}
			}
		}
	}
	
	// Check for negative cycles
	for u := 0; u < g.Vertices; u++ {
		if distances[u] != math.MaxInt32 {
			for _, edge := range g.Edges[u] {
				if distances[u]+edge.Weight < distances[edge.To] {
					return distances, false // Negative cycle detected
				}
			}
		}
	}
	
	return distances, true
}

// TopologicalSort performs topological sorting using DFS
func TopologicalSort(g *Graph) []int {
	visited := make([]bool, g.Vertices)
	stack := make([]int, 0)
	
	for i := 0; i < g.Vertices; i++ {
		if !visited[i] {
			topologicalSortHelper(g, i, visited, &stack)
		}
	}
	
	// Reverse the stack to get topological order
	for i, j := 0, len(stack)-1; i < j; i, j = i+1, j-1 {
		stack[i], stack[j] = stack[j], stack[i]
	}
	
	return stack
}

func topologicalSortHelper(g *Graph, vertex int, visited []bool, stack *[]int) {
	visited[vertex] = true
	
	for _, edge := range g.Edges[vertex] {
		if !visited[edge.To] {
			topologicalSortHelper(g, edge.To, visited, stack)
		}
	}
	
	*stack = append(*stack, vertex)
}

// UnionFind data structure for Kruskal's algorithm
type UnionFind struct {
	parent []int
	rank   []int
}

// NewUnionFind creates a new Union-Find data structure
func NewUnionFind(n int) *UnionFind {
	uf := &UnionFind{
		parent: make([]int, n),
		rank:   make([]int, n),
	}
	
	for i := 0; i < n; i++ {
		uf.parent[i] = i
	}
	
	return uf
}

// Find finds the root of the set containing x
func (uf *UnionFind) Find(x int) int {
	if uf.parent[x] != x {
		uf.parent[x] = uf.Find(uf.parent[x]) // Path compression
	}
	return uf.parent[x]
}

// Union unites the sets containing x and y
func (uf *UnionFind) Union(x, y int) bool {
	rootX := uf.Find(x)
	rootY := uf.Find(y)
	
	if rootX == rootY {
		return false // Already in the same set
	}
	
	// Union by rank
	if uf.rank[rootX] < uf.rank[rootY] {
		uf.parent[rootX] = rootY
	} else if uf.rank[rootX] > uf.rank[rootY] {
		uf.parent[rootY] = rootX
	} else {
		uf.parent[rootY] = rootX
		uf.rank[rootX]++
	}
	
	return true
}

// WeightedEdge represents an edge with weight for MST algorithms
type WeightedEdge struct {
	From   int
	To     int
	Weight int
}

// KruskalMST implements Kruskal's algorithm for Minimum Spanning Tree
func KruskalMST(g *Graph) []WeightedEdge {
	edges := make([]WeightedEdge, 0)
	
	// Collect all edges
	for from := 0; from < g.Vertices; from++ {
		for _, edge := range g.Edges[from] {
			edges = append(edges, WeightedEdge{From: from, To: edge.To, Weight: edge.Weight})
		}
	}
	
	// Sort edges by weight
	for i := 0; i < len(edges)-1; i++ {
		for j := 0; j < len(edges)-i-1; j++ {
			if edges[j].Weight > edges[j+1].Weight {
				edges[j], edges[j+1] = edges[j+1], edges[j]
			}
		}
	}
	
	uf := NewUnionFind(g.Vertices)
	mst := make([]WeightedEdge, 0)
	
	for _, edge := range edges {
		if uf.Union(edge.From, edge.To) {
			mst = append(mst, edge)
			if len(mst) == g.Vertices-1 {
				break
			}
		}
	}
	
	return mst
}

// PrimMST implements Prim's algorithm for Minimum Spanning Tree
func PrimMST(g *Graph) []WeightedEdge {
	if g.Vertices == 0 {
		return nil
	}
	
	inMST := make([]bool, g.Vertices)
	key := make([]int, g.Vertices)
	parent := make([]int, g.Vertices)
	
	// Initialize keys to infinity
	for i := 0; i < g.Vertices; i++ {
		key[i] = math.MaxInt32
		parent[i] = -1
	}
	
	key[0] = 0 // Start from vertex 0
	mst := make([]WeightedEdge, 0)
	
	for count := 0; count < g.Vertices; count++ {
		// Find minimum key vertex not yet in MST
		u := -1
		for v := 0; v < g.Vertices; v++ {
			if !inMST[v] && (u == -1 || key[v] < key[u]) {
				u = v
			}
		}
		
		inMST[u] = true
		
		// Add edge to MST (except for the first vertex)
		if parent[u] != -1 {
			mst = append(mst, WeightedEdge{From: parent[u], To: u, Weight: key[u]})
		}
		
		// Update keys of adjacent vertices
		for _, edge := range g.Edges[u] {
			v := edge.To
			if !inMST[v] && edge.Weight < key[v] {
				key[v] = edge.Weight
				parent[v] = u
			}
		}
	}
	
	return mst
}

// StronglyConnectedComponents finds SCCs using Tarjan's algorithm
func StronglyConnectedComponents(g *Graph) [][]int {
	index := 0
	stack := make([]int, 0)
	indices := make([]int, g.Vertices)
	lowlinks := make([]int, g.Vertices)
	onStack := make([]bool, g.Vertices)
	sccs := make([][]int, 0)
	
	for i := 0; i < g.Vertices; i++ {
		indices[i] = -1
	}
	
	for v := 0; v < g.Vertices; v++ {
		if indices[v] == -1 {
			tarjanSCC(g, v, &index, &stack, indices, lowlinks, onStack, &sccs)
		}
	}
	
	return sccs
}

func tarjanSCC(g *Graph, v int, index *int, stack *[]int, indices, lowlinks []int, onStack []bool, sccs *[][]int) {
	indices[v] = *index
	lowlinks[v] = *index
	*index++
	*stack = append(*stack, v)
	onStack[v] = true
	
	for _, edge := range g.Edges[v] {
		w := edge.To
		if indices[w] == -1 {
			tarjanSCC(g, w, index, stack, indices, lowlinks, onStack, sccs)
			lowlinks[v] = Min(lowlinks[v], lowlinks[w])
		} else if onStack[w] {
			lowlinks[v] = Min(lowlinks[v], indices[w])
		}
	}
	
	// If v is a root node, pop the stack and create an SCC
	if lowlinks[v] == indices[v] {
		scc := make([]int, 0)
		for {
			w := (*stack)[len(*stack)-1]
			*stack = (*stack)[:len(*stack)-1]
			onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		*sccs = append(*sccs, scc)
	}
}

// ArticulationPoints finds articulation points (cut vertices) in the graph
func ArticulationPoints(g *Graph) []int {
	visited := make([]bool, g.Vertices)
	disc := make([]int, g.Vertices)
	low := make([]int, g.Vertices)
	parent := make([]int, g.Vertices)
	ap := make([]bool, g.Vertices)
	time := 0
	
	for i := 0; i < g.Vertices; i++ {
		parent[i] = -1
	}
	
	for i := 0; i < g.Vertices; i++ {
		if !visited[i] {
			apHelper(g, i, visited, disc, low, parent, ap, &time)
		}
	}
	
	result := make([]int, 0)
	for i := 0; i < g.Vertices; i++ {
		if ap[i] {
			result = append(result, i)
		}
	}
	
	return result
}

func apHelper(g *Graph, u int, visited []bool, disc, low, parent []int, ap []bool, time *int) {
	children := 0
	visited[u] = true
	disc[u] = *time
	low[u] = *time
	*time++
	
	for _, edge := range g.Edges[u] {
		v := edge.To
		if !visited[v] {
			children++
			parent[v] = u
			apHelper(g, v, visited, disc, low, parent, ap, time)
			
			low[u] = Min(low[u], low[v])
			
			// u is an articulation point in the following cases:
			// (1) u is root of DFS tree and has two or more children
			if parent[u] == -1 && children > 1 {
				ap[u] = true
			}
			
			// (2) u is not root and low value of one of its children is more than or equal to discovery value of u
			if parent[u] != -1 && low[v] >= disc[u] {
				ap[u] = true
			}
		} else if v != parent[u] {
			low[u] = Min(low[u], disc[v])
		}
	}
}