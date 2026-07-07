package domain

import (
	pkgStorage "github.com/GustavoCaso/expensetrace/internal/storage"
)

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
	PreviewExpenses []pkgStorage.Expense
	TotalRows       int
	Errors          []string
}
