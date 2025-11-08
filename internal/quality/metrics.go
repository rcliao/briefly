package quality

import (
	"regexp"
	"strings"
)

// DigestQualityMetrics contains all quality metrics for a digest
type DigestQualityMetrics struct {
	// Article coverage
	ArticleCount   int     `json:"article_count"`
	CitationsFound int     `json:"citations_found"`
	CoveragePct    float64 `json:"coverage_pct"`

	// Vagueness detection
	VaguePhrases     int      `json:"vague_phrases"`
	VaguePhrasesList []string `json:"vague_phrases_list,omitempty"`

	// Specificity checks
	WordCount        int  `json:"word_count"`
	HasNumbers       bool `json:"has_numbers"`
	NumberCount      int  `json:"number_count"`
	HasProperNouns   bool `json:"has_proper_nouns"`
	ProperNounCount  int  `json:"proper_noun_count"`
	SpecificityScore int  `json:"specificity_score"` // 0-100

	// Citation analysis
	CitationDensity float64 `json:"citation_density"` // Citations per 100 words
	UncitedArticles []int   `json:"uncited_articles,omitempty"`

	// Quality assessment
	Grade    string   `json:"grade"` // A/B/C/D
	Warnings []string `json:"warnings"`
	Passed   bool     `json:"passed"` // Overall pass/fail
}

// ClusterCoherenceMetrics contains quality metrics for topic clustering
type ClusterCoherenceMetrics struct {
	// Cluster count
	NumClusters    int     `json:"num_clusters"`
	NumArticles    int     `json:"num_articles"`
	AvgClusterSize float64 `json:"avg_cluster_size"`

	// Silhouette scores (range: -1 to 1, higher is better)
	AvgSilhouette      float64   `json:"avg_silhouette"`      // Overall clustering quality
	ClusterSilhouettes []float64 `json:"cluster_silhouettes"` // Per-cluster scores

	// Cohesion metrics (0 to 1, higher is better)
	AvgIntraClusterSimilarity float64   `json:"avg_intra_cluster_similarity"` // Avg similarity within clusters
	IntraClusterSimilarities  []float64 `json:"intra_cluster_similarities"`   // Per-cluster cohesion

	// Separation metrics (0 to 1, higher is better)
	AvgInterClusterDistance float64 `json:"avg_inter_cluster_distance"` // Distance between cluster centroids

	// Quality assessment
	CoherenceGrade string   `json:"coherence_grade"` // A/B/C/D
	Issues         []string `json:"issues,omitempty"`
	Passed         bool     `json:"passed"`
}

// QualityThresholds defines minimum acceptable quality levels
type QualityThresholds struct {
	// Digest quality
	MinCoveragePct      float64 `yaml:"min_coverage_pct"`      // Default: 0.80 (80%)
	MaxVaguePhrases     int     `yaml:"max_vague_phrases"`     // Default: 2
	MinWordCount        int     `yaml:"min_word_count"`        // Default: 150
	MaxWordCount        int     `yaml:"max_word_count"`        // Default: 400
	MinSpecificityScore int     `yaml:"min_specificity_score"` // Default: 50
	MinCitationDensity  float64 `yaml:"min_citation_density"`  // Default: 2.0 (2 citations per 100 words)

	// Cluster quality
	MinSilhouetteScore  float64 `yaml:"min_silhouette_score"`   // Default: 0.3
	MinIntraClusterSim  float64 `yaml:"min_intra_cluster_sim"`  // Default: 0.5
	MinInterClusterDist float64 `yaml:"min_inter_cluster_dist"` // Default: 0.3

	// Grade cutoffs
	GradeAThresholds GradeThresholds `yaml:"grade_a"`
	GradeBThresholds GradeThresholds `yaml:"grade_b"`
	GradeCThresholds GradeThresholds `yaml:"grade_c"`
}

// GradeThresholds defines requirements for each grade level
type GradeThresholds struct {
	MinCoverage    float64 `yaml:"min_coverage"`
	MaxVague       int     `yaml:"max_vague"`
	MinSpecificity int     `yaml:"min_specificity"`
	RequireNumbers bool    `yaml:"require_numbers"`
	RequireNames   bool    `yaml:"require_names"`
	MinSilhouette  float64 `yaml:"min_silhouette"`
}

// DefaultThresholds returns the default quality thresholds
func DefaultThresholds() QualityThresholds {
	return QualityThresholds{
		MinCoveragePct:      0.80,
		MaxVaguePhrases:     2,
		MinWordCount:        150,
		MaxWordCount:        400,
		MinSpecificityScore: 50,
		MinCitationDensity:  2.0,
		MinSilhouetteScore:  0.3,
		MinIntraClusterSim:  0.5,
		MinInterClusterDist: 0.3,

		GradeAThresholds: GradeThresholds{
			MinCoverage:    0.90,
			MaxVague:       1,
			MinSpecificity: 70,
			RequireNumbers: true,
			RequireNames:   true,
			MinSilhouette:  0.5,
		},
		GradeBThresholds: GradeThresholds{
			MinCoverage:    0.80,
			MaxVague:       2,
			MinSpecificity: 50,
			RequireNumbers: true,
			RequireNames:   false,
			MinSilhouette:  0.4,
		},
		GradeCThresholds: GradeThresholds{
			MinCoverage:    0.60,
			MaxVague:       3,
			MinSpecificity: 30,
			RequireNumbers: false,
			RequireNames:   false,
			MinSilhouette:  0.3,
		},
	}
}

// VaguePhrases returns the list of generic phrases to detect
var VaguePhrases = []string{
	"several",
	"various",
	"multiple",
	"many",
	"some",
	"a number of",
	"numerous",
	"different",
	"various sources",
	"a few",
	"a couple of",
	"certain",
}

// DetectVaguePhrases counts and lists vague phrases in text
func DetectVaguePhrases(text string) (count int, found []string) {
	textLower := strings.ToLower(text)
	foundMap := make(map[string]bool)

	for _, phrase := range VaguePhrases {
		if strings.Contains(textLower, phrase) {
			if !foundMap[phrase] {
				foundMap[phrase] = true
				found = append(found, phrase)
			}
			// Count all occurrences
			count += strings.Count(textLower, phrase)
		}
	}

	return count, found
}

// DetectNumbers finds specific numbers, percentages, and metrics in text
func DetectNumbers(text string) (count int, hasNumbers bool) {
	// Regex for numbers: percentages, dollar amounts, numbers with commas, multipliers
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\d+%`),                     // 40%
		regexp.MustCompile(`\d+x`),                     // 10x
		regexp.MustCompile(`\$[\d,]+(?:\.\d+)?[BMK]?`), // $1.5M, $100K
		regexp.MustCompile(`[\d,]+`),                   // 1,000 or 42
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllString(text, -1)
		count += len(matches)
	}

	hasNumbers = count > 0
	return count, hasNumbers
}

// DetectProperNouns finds capitalized names (rough heuristic)
func DetectProperNouns(text string) (count int, hasNames bool) {
	// Regex for proper nouns: capitalized words (but not at start of sentences)
	// This is a heuristic - will match company names, people, etc.
	pattern := regexp.MustCompile(`\b[A-Z][a-z]+\s+[A-Z][a-z]+\b`)
	matches := pattern.FindAllString(text, -1)

	// Also match single capitalized words that are likely names/companies
	singlePattern := regexp.MustCompile(`\b[A-Z][a-z]{2,}\b`)
	singleMatches := singlePattern.FindAllString(text, -1)

	// Combine and deduplicate
	nameMap := make(map[string]bool)
	for _, match := range matches {
		nameMap[match] = true
	}
	for _, match := range singleMatches {
		// Skip common words that aren't names
		if !isCommonWord(match) {
			nameMap[match] = true
		}
	}

	count = len(nameMap)
	hasNames = count > 0
	return count, hasNames
}

// isCommonWord checks if a capitalized word is likely a common word (not a name)
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"The": true, "This": true, "That": true, "These": true, "Those": true,
		"While": true, "When": true, "Where": true, "What": true, "Which": true,
		"However": true, "Although": true, "Despite": true, "Through": true,
		"After": true, "Before": true, "During": true, "Since": true,
	}
	return commonWords[word]
}

// ExtractCitations finds all citation numbers in text [1] [2] [3]
func ExtractCitations(text string) []int {
	pattern := regexp.MustCompile(`\[(\d+)\]`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	citationMap := make(map[int]bool)
	citations := []int{}

	for _, match := range matches {
		if len(match) > 1 {
			// Convert to int (match[1] is guaranteed to be digits from the regex pattern)
			citationNum := 0
			for _, c := range match[1] {
				citationNum = citationNum*10 + int(c-'0')
			}
			if !citationMap[citationNum] {
				citationMap[citationNum] = true
				citations = append(citations, citationNum)
			}
		}
	}

	return citations
}

// CalculateSpecificityScore computes an overall specificity score (0-100)
func CalculateSpecificityScore(numberCount, properNounCount, vaguePhrases int) int {
	score := 0

	// Numbers contribute up to 40 points (10 points per number, max 4)
	score += min(numberCount*10, 40)

	// Proper nouns contribute up to 40 points (8 points per name, max 5)
	score += min(properNounCount*8, 40)

	// Vague phrases penalize: -10 points each
	score -= vaguePhrases * 10

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// GradeDigestQuality assigns a letter grade based on metrics
func GradeDigestQuality(metrics *DigestQualityMetrics, thresholds QualityThresholds) string {
	// Check Grade A
	if metrics.CoveragePct >= thresholds.GradeAThresholds.MinCoverage &&
		metrics.VaguePhrases <= thresholds.GradeAThresholds.MaxVague &&
		metrics.SpecificityScore >= thresholds.GradeAThresholds.MinSpecificity &&
		(!thresholds.GradeAThresholds.RequireNumbers || metrics.HasNumbers) &&
		(!thresholds.GradeAThresholds.RequireNames || metrics.HasProperNouns) {
		return "A - EXCELLENT"
	}

	// Check Grade B
	if metrics.CoveragePct >= thresholds.GradeBThresholds.MinCoverage &&
		metrics.VaguePhrases <= thresholds.GradeBThresholds.MaxVague &&
		metrics.SpecificityScore >= thresholds.GradeBThresholds.MinSpecificity &&
		(!thresholds.GradeBThresholds.RequireNumbers || metrics.HasNumbers) {
		return "B - GOOD"
	}

	// Check Grade C
	if metrics.CoveragePct >= thresholds.GradeCThresholds.MinCoverage &&
		metrics.VaguePhrases <= thresholds.GradeCThresholds.MaxVague &&
		metrics.SpecificityScore >= thresholds.GradeCThresholds.MinSpecificity {
		return "C - FAIR"
	}

	// Grade D (poor quality)
	return "D - POOR"
}

// GradeClusterCoherence assigns a letter grade based on coherence metrics
func GradeClusterCoherence(metrics *ClusterCoherenceMetrics, thresholds QualityThresholds) string {
	// Check Grade A - Excellent clustering
	if metrics.AvgSilhouette >= thresholds.GradeAThresholds.MinSilhouette &&
		metrics.AvgIntraClusterSimilarity >= thresholds.MinIntraClusterSim+0.1 {
		return "A - EXCELLENT"
	}

	// Check Grade B - Good clustering
	if metrics.AvgSilhouette >= thresholds.GradeBThresholds.MinSilhouette &&
		metrics.AvgIntraClusterSimilarity >= thresholds.MinIntraClusterSim {
		return "B - GOOD"
	}

	// Check Grade C - Fair clustering
	if metrics.AvgSilhouette >= thresholds.GradeCThresholds.MinSilhouette {
		return "C - FAIR"
	}

	// Grade D - Poor clustering
	return "D - POOR"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
