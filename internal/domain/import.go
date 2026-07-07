package domain

// PreviewData is the view model rendered by the import preview partial.
type PreviewData struct {
	ViewBase
	ImportSessionID string
	Filename        string
	Headers         []string
	PreviewRows     [][]string
	TotalRows       int
}

// MappingData is the view model rendered by the import mapping-preview partial.
type MappingData struct {
	ViewBase
	ImportSessionID string
	Headers         []string
	PreviewExpenses []Expense
	TotalRows       int
	Errors          []string
}
