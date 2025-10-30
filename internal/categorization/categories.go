package categorization

// Category represents an article category with metadata
type Category struct {
	Name        string
	Icon        string
	Description string
	Priority    int // Lower number = higher priority in display
}

// DefaultCategories returns the standard category set
func DefaultCategories() []Category {
	return []Category{
		{
			Name:        "Platform Updates",
			Icon:        "📦",
			Description: "Product launches, feature releases, and platform announcements",
			Priority:    1,
		},
		{
			Name:        "From the Field",
			Icon:        "💭",
			Description: "Practitioner insights, workflows, and real-world experiences",
			Priority:    2,
		},
		{
			Name:        "Research",
			Icon:        "📊",
			Description: "Academic papers, studies, and research findings",
			Priority:    3,
		},
		{
			Name:        "Tutorials",
			Icon:        "🎓",
			Description: "How-to guides, walkthroughs, and educational content",
			Priority:    4,
		},
		{
			Name:        "Analysis",
			Icon:        "🔍",
			Description: "Deep dives, opinion pieces, and analytical content",
			Priority:    5,
		},
		{
			Name:        "Miscellaneous",
			Icon:        "📌",
			Description: "Other noteworthy content",
			Priority:    99,
		},
	}
}

// GetCategoryByName returns a category by its name, or nil if not found
func GetCategoryByName(name string, categories []Category) *Category {
	for _, cat := range categories {
		if cat.Name == name {
			return &cat
		}
	}
	return nil
}

// GetCategoryNames returns just the category names
func GetCategoryNames(categories []Category) []string {
	names := make([]string, len(categories))
	for i, cat := range categories {
		names[i] = cat.Name
	}
	return names
}
