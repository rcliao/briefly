package tools

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// citationPattern matches [N] citation references in text.
var citationPattern = regexp.MustCompile(`\[(\d+)\]`)

// remapCitations replaces all [N] references in text using the provided mapping.
// localToGlobal maps local citation numbers (1-based) to global citation numbers.
// For example, if localToGlobal = {1: 5, 2: 3}, then [1] becomes [5] and [2] becomes [3].
func remapCitations(text string, localToGlobal map[int]int) string {
	if len(localToGlobal) == 0 || text == "" {
		return text
	}

	// We need to replace in a way that avoids double-replacement.
	// Strategy: replace [N] with a unique placeholder first, then replace placeholders.
	type replacement struct {
		old string
		new string
	}

	// Find all unique citation numbers in the text
	matches := citationPattern.FindAllStringSubmatch(text, -1)
	seen := make(map[int]bool)
	var replacements []replacement

	for _, match := range matches {
		localNum, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		if seen[localNum] {
			continue
		}
		seen[localNum] = true

		if globalNum, ok := localToGlobal[localNum]; ok {
			placeholder := fmt.Sprintf("<<CITE_%d>>", localNum)
			replacements = append(replacements, replacement{
				old: fmt.Sprintf("[%d]", localNum),
				new: placeholder,
			})
			// Second pass: placeholder -> final
			replacements = append(replacements, replacement{
				old: placeholder,
				new: fmt.Sprintf("[%d]", globalNum),
			})
		}
	}

	// Apply replacements in two passes to avoid conflicts
	result := text
	// Pass 1: [N] -> placeholder
	for i := 0; i < len(replacements); i += 2 {
		result = strings.ReplaceAll(result, replacements[i].old, replacements[i].new)
	}
	// Pass 2: placeholder -> [global]
	for i := 1; i < len(replacements); i += 2 {
		result = strings.ReplaceAll(result, replacements[i].old, replacements[i].new)
	}

	return result
}

// remapCitationSlice remaps citations in a slice of strings.
func remapCitationSlice(items []string, localToGlobal map[int]int) []string {
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = remapCitations(item, localToGlobal)
	}
	return result
}

// remapArticleRefs converts local article ref numbers to global citation numbers.
func remapArticleRefs(refs []int, localToGlobal map[int]int) []int {
	result := make([]int, 0, len(refs))
	for _, ref := range refs {
		if global, ok := localToGlobal[ref]; ok {
			result = append(result, global)
		} else {
			result = append(result, ref) // Keep original if no mapping
		}
	}
	return result
}

// buildClusterCitationMap builds a mapping from cluster-local article numbers (1-based)
// to global citation numbers, given the cluster's article IDs and a function to look up global numbers.
func buildClusterCitationMap(articleIDs []string, getCitationNum func(string) int) map[int]int {
	m := make(map[int]int)
	for i, id := range articleIDs {
		localNum := i + 1 // Generator numbers articles [1], [2], [3]...
		globalNum := getCitationNum(id)
		if globalNum > 0 {
			m[localNum] = globalNum
		}
	}
	return m
}

