package domain

import "github.com/GustavoCaso/expensetrace/internal/storage"

// EnhancedCategory extends storage.Category with extra UI-friendly fields.
type EnhancedCategory struct {
	storage.Category
	AvgAmount       int64
	LastTransaction string
	Total           int
	TotalAmount     int64
	SpendingCount   int
	IncomeCount     int
}

// CategoryFormData holds parsed and validated category form data.
type CategoryFormData struct {
	Name          string
	Pattern       string
	MonthlyBudget int64
}

type CategoriesViewData struct {
	ViewBase
	Categories         []EnhancedCategory
	CategorizedCount   int
	UncategorizedCount int
}

type CategoryViewData struct {
	ViewBase
	Category storage.Category
	Action   string
}

type UncategorizedInfo struct {
	Count    int
	Expenses []storage.Expense
	Total    int64
	Slug     string
}

type UncategorizedViewData struct {
	ViewBase
	Keys             []string
	UncategorizeInfo map[string]UncategorizedInfo
	Categories       []storage.Category
	TotalExpenses    int
	TotalAmount      int64
}

type CreateCategoryViewData struct {
	ViewBase
	Category storage.Category
	Results  []storage.Expense
	Total    int
	Action   string
}
