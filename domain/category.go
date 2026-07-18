package domain

type Category interface {
	ID() int64
	Name() string
	Pattern() string
	MonthlyBudget() int64
}

const ExcludeCategory = "🚫 Exclude"

// EnhancedCategory extends Category with extra UI-friendly fields.
type EnhancedCategory struct {
	Category
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
	Category Category
	Action   string
}

type UncategorizedInfo struct {
	Count    int
	Expenses []Expense
	Total    int64
	Slug     string
}

type UncategorizedViewData struct {
	ViewBase
	Keys             []string
	UncategorizeInfo map[string]UncategorizedInfo
	Categories       []Category
	TotalExpenses    int
	TotalAmount      int64
}

type CreateCategoryViewData struct {
	ViewBase
	Category Category
	Results  []Expense
	Total    int
	Action   string
}

type category struct {
	id            int64
	name          string
	pattern       string
	monthlyBudget int64
}

func (c category) ID() int64 {
	return c.id
}

func (c category) Name() string {
	return c.name
}

func (c category) Pattern() string {
	return c.pattern
}

func (c category) MonthlyBudget() int64 {
	return c.monthlyBudget
}

func NewCategory(id int64, name, pattern string, monthlyBudget int64) Category {
	return category{
		id:            id,
		name:          name,
		pattern:       pattern,
		monthlyBudget: monthlyBudget,
	}
}

func EmptyCategory() Category {
	return NewCategory(0, "", "", 0)
}
