package domain

import "fmt"

type Category interface {
	ID() int64
	Name() string
	Pattern() string
	MonthlyBudget() int64
	Type() CategoryType
}

const ExcludeCategory = "🚫 Exclude"

// CategoryType says which kind of expense a category can hold. Values are
// persisted in the categories.type column, so do not reorder them.
type CategoryType int

const (
	// ChargeCategory holds charges (spending).
	ChargeCategory CategoryType = iota
	// IncomeCategory holds income.
	IncomeCategory
	// AnyCategory holds both; only used by the exclude category.
	AnyCategory
)

// Accepts reports whether an expense of type t can belong to a category of
// this type.
func (c CategoryType) Accepts(t ExpenseType) bool {
	switch c {
	case ChargeCategory:
		return t == ChargeType
	case IncomeCategory:
		return t == IncomeType
	case AnyCategory:
		return true
	}
	return false
}

func (c CategoryType) String() string {
	switch c {
	case ChargeCategory:
		return "charge"
	case IncomeCategory:
		return "income"
	case AnyCategory:
		return "any"
	}
	return "unknown"
}

// ParseCategoryType parses the user-selectable category types ("charge" or
// "income"). An empty string defaults to charge.
func ParseCategoryType(s string) (CategoryType, error) {
	switch s {
	case "", "charge":
		return ChargeCategory, nil
	case "income":
		return IncomeCategory, nil
	}
	return ChargeCategory, fmt.Errorf("invalid category type %q", s)
}

// EnhancedCategory extends Category with extra UI-friendly fields.
type EnhancedCategory struct {
	Category
	AvgAmount       int64
	LastTransaction string
	Total           int
	TotalAmount     int64
	SpendingCount   int
	IncomeCount     int
	// MonthAmount is the signed sum of this month's expenses (charges are
	// negative).
	MonthAmount int64
	// Budget is this month's budget status; Status is no_budget for income
	// categories and charge categories without a budget.
	Budget BudgetInfo
}

// BudgetBarWidth is the budget percentage used, capped at 100, for drawing a
// bar.
func (c EnhancedCategory) BudgetBarWidth() float64 {
	return min(c.Budget.PercentageUsed, percentageCap)
}

// CategoryFormData holds parsed and validated category form data.
type CategoryFormData struct {
	Name          string
	Pattern       string
	MonthlyBudget int64
	Type          CategoryType
}

type CategoriesViewData struct {
	ViewBase
	// ExpenseCategories and IncomeCategories are sorted by name; Excluded is
	// the exclude category (nil if missing).
	ExpenseCategories  []EnhancedCategory
	IncomeCategories   []EnhancedCategory
	Excluded           *EnhancedCategory
	CategorizedCount   int
	UncategorizedCount int
}

type CategoryViewData struct {
	ViewBase
	Category Category
	Action   string
}

// UncategorizedInfo groups uncategorized expenses sharing a description and
// type; a description seen as both a charge and income is two groups.
type UncategorizedInfo struct {
	Description string
	Type        ExpenseType
	Count       int
	Expenses    []Expense
	Total       int64
}

type UncategorizedViewData struct {
	ViewBase
	Groups        []UncategorizedInfo
	Categories    []Category
	TotalExpenses int
	TotalAmount   int64
	ChargeGroups  int
	IncomeGroups  int
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
	categoryType  CategoryType
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

func (c category) Type() CategoryType {
	return c.categoryType
}

func NewCategory(id int64, name, pattern string, monthlyBudget int64, categoryType CategoryType) Category {
	return category{
		id:            id,
		name:          name,
		pattern:       pattern,
		monthlyBudget: monthlyBudget,
		categoryType:  categoryType,
	}
}

func EmptyCategory() Category {
	return NewCategory(0, "", "", 0, ChargeCategory)
}
