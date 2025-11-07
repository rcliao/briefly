package clustering

import (
	"fmt"
	"reflect"

	"github.com/humilityai/hdbscan"
)

// InspectHDBSCANClustering is a helper to inspect the Clustering struct at runtime
func InspectHDBSCANClustering() {
	// Create sample data
	data := [][]float64{
		{1.0, 2.0, 3.0},
		{1.1, 2.1, 3.1},
		{1.2, 2.2, 3.2},
		{5.0, 6.0, 7.0},
		{5.1, 6.1, 7.1},
		{5.2, 6.2, 7.2},
		{10.0, 11.0, 12.0},
		{10.1, 11.1, 12.1},
	}

	// Create clustering
	clustering, err := hdbscan.NewClustering(data, 2)
	if err != nil {
		fmt.Printf("Error creating clustering: %v\n", err)
		return
	}

	// Run clustering
	err = clustering.Run(hdbscan.EuclideanDistance, hdbscan.VarianceScore, true)
	if err != nil {
		fmt.Printf("Error running clustering: %v\n", err)
		return
	}

	fmt.Println("=== HDBSCAN Clustering Inspection ===")
	fmt.Println()

	// Use reflection to inspect the struct
	v := reflect.ValueOf(clustering).Elem()
	t := v.Type()

	fmt.Println("Clustering struct fields:")
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		fmt.Printf("  Field %d: %s (Type: %s)\n", i, field.Name, field.Type)

		// If it's the Clusters field, inspect it further
		if field.Name == "Clusters" {
			fmt.Println("  Inspecting Clusters field...")
			inspectClusters(value)
		}
	}
}

func inspectClusters(clustersValue reflect.Value) {
	// Check if it's a map
	if clustersValue.Kind() == reflect.Map {
		fmt.Printf("    Clusters is a Map with %d entries\n", clustersValue.Len())
		keys := clustersValue.MapKeys()
		for i, key := range keys {
			if i < 3 { // Show first 3 entries
				val := clustersValue.MapIndex(key)
				fmt.Printf("      Key: %v, Value Type: %s\n", key, val.Type())

				// Try to inspect the value
				if val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr {
					val = val.Elem()
				}

				if val.Kind() == reflect.Struct {
					fmt.Printf("        Struct with %d fields\n", val.NumField())
					inspectStruct(val, 8)
				} else if val.Kind() == reflect.Slice {
					fmt.Printf("        Slice with %d elements\n", val.Len())
					if val.Len() > 0 {
						fmt.Printf("          Element type: %s\n", val.Index(0).Type())
					}
				}
			}
		}
	} else if clustersValue.Kind() == reflect.Slice {
		fmt.Printf("    Clusters is a Slice with %d elements\n", clustersValue.Len())
		if clustersValue.Len() > 0 {
			fmt.Printf("      Element type: %s\n", clustersValue.Index(0).Type())
			// Inspect first element
			inspectStruct(clustersValue.Index(0), 6)
		}
	} else {
		fmt.Printf("    Clusters is of type: %s\n", clustersValue.Type())
	}
}

func inspectStruct(v reflect.Value, indent int) {
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += " "
	}

	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Only show exported fields
		if field.PkgPath != "" {
			continue
		}

		fmt.Printf("%sField: %s (Type: %s)\n", indentStr, field.Name, field.Type)

		// If it's a slice or map, show length
		if value.Kind() == reflect.Slice || value.Kind() == reflect.Map {
			fmt.Printf("%s  Length: %d\n", indentStr, value.Len())
		}
	}
}
