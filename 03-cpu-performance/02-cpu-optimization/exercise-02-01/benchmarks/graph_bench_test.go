package benchmarks

import (
	"fmt"
	"testing"

	"exercise-02-01/algorithms"
)

// Graph benchmark sizes (smaller due to complexity)
var graphSizes = []int{10, 50, 100, 200}

// BenchmarkDepthFirstSearch benchmarks DFS algorithm
func BenchmarkDepthFirstSearch(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*2)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.DepthFirstSearch(graph, 0)
			}
		})
	}
}

// BenchmarkBreadthFirstSearch benchmarks BFS algorithm
func BenchmarkBreadthFirstSearch(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*2)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.BreadthFirstSearch(graph, 0)
			}
		})
	}
}

// BenchmarkDijkstra benchmarks Dijkstra's shortest path algorithm
func BenchmarkDijkstra(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*3)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.Dijkstra(graph, 0)
			}
		})
	}
}

// BenchmarkFloydWarshall benchmarks Floyd-Warshall all-pairs shortest path
func BenchmarkFloydWarshall(b *testing.B) {
	// Smaller sizes due to O(V³) complexity
	sizes := []int{10, 25, 50, 75}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*2)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.FloydWarshall(graph)
			}
		})
	}
}

// BenchmarkBellmanFord benchmarks Bellman-Ford algorithm
func BenchmarkBellmanFord(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*2)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.BellmanFord(graph, 0)
			}
		})
	}
}

// BenchmarkTopologicalSort benchmarks topological sorting
func BenchmarkTopologicalSort(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			// Generate a DAG (Directed Acyclic Graph) for topological sort
			graph := generateDAG(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.TopologicalSort(graph)
			}
		})
	}
}

// BenchmarkKruskalMST benchmarks Kruskal's MST algorithm
func BenchmarkKruskalMST(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*3)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.KruskalMST(graph)
			}
		})
	}
}

// BenchmarkPrimMST benchmarks Prim's MST algorithm
func BenchmarkPrimMST(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*3)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.PrimMST(graph)
			}
		})
	}
}

// BenchmarkStronglyConnectedComponents benchmarks Tarjan's SCC algorithm
func BenchmarkStronglyConnectedComponents(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*2)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.StronglyConnectedComponents(graph)
			}
		})
	}
}

// BenchmarkArticulationPoints benchmarks articulation points algorithm
func BenchmarkArticulationPoints(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*2)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.ArticulationPoints(graph)
			}
		})
	}
}

// Graph density benchmarks

// BenchmarkDijkstraSparseGraph benchmarks Dijkstra on sparse graphs
func BenchmarkDijkstraSparseGraph(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			// Sparse graph: E = V
			graph := algorithms.GenerateRandomGraph(size, size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.Dijkstra(graph, 0)
			}
		})
	}
}

// BenchmarkDijkstraDenseGraph benchmarks Dijkstra on dense graphs
func BenchmarkDijkstraDenseGraph(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			// Dense graph: E = V²/2
			graph := algorithms.GenerateRandomGraph(size, size*size/2)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.Dijkstra(graph, 0)
			}
		})
	}
}

// Traversal comparison benchmarks

// BenchmarkTraversalComparison compares DFS and BFS
func BenchmarkTraversalComparison(b *testing.B) {
	size := 100
	graph := algorithms.GenerateRandomGraph(size, size*2)
	
	b.Run("DFS", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.DepthFirstSearch(graph, 0)
		}
	})
	
	b.Run("BFS", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.BreadthFirstSearch(graph, 0)
		}
	})
}

// BenchmarkShortestPathComparison compares shortest path algorithms
func BenchmarkShortestPathComparison(b *testing.B) {
	size := 50
	graph := algorithms.GenerateRandomGraph(size, size*2)
	
	b.Run("Dijkstra", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.Dijkstra(graph, 0)
		}
	})
	
	b.Run("BellmanFord", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.BellmanFord(graph, 0)
		}
	})
	
	b.Run("FloydWarshall", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.FloydWarshall(graph)
		}
	})
}

// BenchmarkMSTComparison compares MST algorithms
func BenchmarkMSTComparison(b *testing.B) {
	size := 100
	graph := algorithms.GenerateRandomGraph(size, size*3)
	
	b.Run("Kruskal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.KruskalMST(graph)
		}
	})
	
	b.Run("Prim", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.PrimMST(graph)
		}
	})
}

// Memory usage benchmarks

// BenchmarkGraphMemoryUsage measures memory allocations for graph algorithms
func BenchmarkGraphMemoryUsage(b *testing.B) {
	size := 100
	graph := algorithms.GenerateRandomGraph(size, size*2)
	
	b.Run("DFS", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.DepthFirstSearch(graph, 0)
		}
	})
	
	b.Run("BFS", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.BreadthFirstSearch(graph, 0)
		}
	})
	
	b.Run("Dijkstra", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.Dijkstra(graph, 0)
		}
	})
	
	b.Run("FloydWarshall", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.FloydWarshall(graph)
		}
	})
}

// Union-Find benchmarks

// BenchmarkUnionFind benchmarks Union-Find operations
func BenchmarkUnionFind(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				uf := algorithms.NewUnionFind(size)
				
				// Perform random union operations
				for j := 0; j < size/2; j++ {
					x := j % size
					y := (j + 1) % size
					uf.Union(x, y)
				}
				
				// Perform find operations
				for j := 0; j < size; j++ {
					uf.Find(j)
				}
			}
		})
	}
}

// Graph construction benchmarks

// BenchmarkGraphConstruction benchmarks graph creation
func BenchmarkGraphConstruction(b *testing.B) {
	for _, size := range graphSizes {
		b.Run(fmt.Sprintf("Vertices%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				algorithms.GenerateRandomGraph(size, size*2)
			}
		})
	}
}

// Scalability benchmarks

// BenchmarkGraphScalability tests algorithm scalability
func BenchmarkGraphScalability(b *testing.B) {
	largeSizes := []int{500, 1000, 2000}
	
	for _, size := range largeSizes {
		b.Run(fmt.Sprintf("DFS_Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*2)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.DepthFirstSearch(graph, 0)
			}
		})
		
		b.Run(fmt.Sprintf("BFS_Vertices%d", size), func(b *testing.B) {
			graph := algorithms.GenerateRandomGraph(size, size*2)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.BreadthFirstSearch(graph, 0)
			}
		})
	}
}

// Helper functions

// generateDAG creates a Directed Acyclic Graph for topological sort testing
func generateDAG(vertices int) *algorithms.Graph {
	graph := algorithms.NewGraph(vertices)
	
	// Add edges only from lower to higher numbered vertices to ensure no cycles
	for i := 0; i < vertices; i++ {
		for j := i + 1; j < vertices; j++ {
			// Add edge with some probability
			if (i*j)%3 == 0 && j-i <= vertices/4 {
				weight := (i + j) % 100 + 1
				graph.AddEdge(i, j, weight)
			}
		}
	}
	
	return graph
}