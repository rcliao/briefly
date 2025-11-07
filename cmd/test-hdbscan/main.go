package main

import (
	"briefly/internal/clustering"
	"fmt"

	"github.com/humilityai/hdbscan"
)

func main() {
	fmt.Println("=== HDBSCAN Cluster Extraction Test ===")
	fmt.Println()

	// Create sample data with 3 well-separated groups (5 points each)
	data := [][]float64{
		// Group 1: Close to (0, 0, 0)
		{0.0, 0.0, 0.0}, {0.1, 0.1, 0.1}, {0.2, 0.2, 0.2}, {-0.1, -0.1, -0.1}, {0.15, 0.15, 0.15},
		// Group 2: Close to (10, 10, 10)
		{10.0, 10.0, 10.0}, {10.1, 10.1, 10.1}, {10.2, 10.2, 10.2}, {9.9, 9.9, 9.9}, {10.15, 10.15, 10.15},
		// Group 3: Close to (20, 20, 20)
		{20.0, 20.0, 20.0}, {20.1, 20.1, 20.1}, {20.2, 20.2, 20.2}, {19.9, 19.9, 19.9}, {20.15, 20.15, 20.15},
		// Outliers
		{100.0, 100.0, 100.0}, {-50.0, -50.0, -50.0},
	}

	fmt.Printf("Input: %d data points (3 groups of 5 + 2 outliers)\n\n", len(data))

	// Create clustering with min_cluster_size=3 (to match our group sizes)
	c, err := hdbscan.NewClustering(data, 3)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Configure and run
	c = c.OutlierDetection().Verbose()
	err = c.Run(hdbscan.EuclideanDistance, hdbscan.VarianceScore, true)
	if err != nil {
		fmt.Printf("Error running: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("=== Extracting Cluster Data ===")
	fmt.Println()

	// Test our extraction function
	clusterData := clustering.ExtractClusterDataPublic(c)

	fmt.Printf("Found %d clusters:\n\n", len(clusterData))

	for i, cluster := range clusterData {
		fmt.Printf("Cluster %d:\n", i)
		fmt.Printf("  Points: %v (count: %d)\n", cluster.Points, len(cluster.Points))
		fmt.Printf("  Centroid: [%.2f, %.2f, %.2f]\n",
			cluster.Centroid[0], cluster.Centroid[1], cluster.Centroid[2])
		fmt.Println()
	}

	// Calculate noise
	totalAssigned := 0
	for _, cluster := range clusterData {
		totalAssigned += len(cluster.Points)
	}
	noiseCount := len(data) - totalAssigned

	fmt.Printf("Total points: %d\n", len(data))
	fmt.Printf("Assigned to clusters: %d\n", totalAssigned)
	fmt.Printf("Noise/Outliers: %d\n", noiseCount)

	if noiseCount == 2 {
		fmt.Println("\n✅ Success! Correctly identified 2 outliers")
	} else {
		fmt.Printf("⚠️  Expected 2 outliers, found %d\n", noiseCount)
	}

	if len(clusterData) == 3 {
		fmt.Println("✅ Success! Correctly found 3 clusters")
	} else {
		fmt.Printf("⚠️  Expected 3 clusters, found %d\n", len(clusterData))
	}
}
