package router

const (
	pageReports    = "reports"
	pageExpenses   = "expenses"
	pageCategories = "categories"
	pageImport     = "import"
)

type banner struct {
	Icon    string
	Message string
}

type viewBase struct {
	Error       string
	Banner      banner
	CurrentPage string
}
