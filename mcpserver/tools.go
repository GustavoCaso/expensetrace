package mcpserver

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GustavoCaso/expensetrace/domain"
)

const (
	dateLayout  = "2006-01-02"
	monthLayout = "2006-01"

	defaultExpenseLimit       = 50
	maxExpenseLimit           = 500
	defaultUncategorizedLimit = 25
	maxUncategorizedLimit     = 200
	defaultTrendMonths        = 12
	maxTrendMonths            = 36

	centsPerUnit = 100

	typeCharge = "charge"
	typeIncome = "income"
)

func (s *server) registerTools(mcpServer *mcp.Server) {
	addTool(s, mcpServer, &mcp.Tool{
		Name:  "list_expenses",
		Title: "List expenses",
		Description: "List the user's transactions, newest first by default. All filters are optional and " +
			"combine with AND. Results are capped by limit; check truncated and total_matching to know " +
			"whether more exist.",
	}, s.listExpenses)

	addTool(s, mcpServer, &mcp.Tool{
		Name:        "list_categories",
		Title:       "List categories",
		Description: "List the user's categories with their matching regex pattern and monthly budget.",
	}, s.listCategories)

	addTool(s, mcpServer, &mcp.Tool{
		Name:  "get_monthly_report",
		Title: "Get monthly report",
		Description: "Income, spending and savings for one month, broken down by category with budget status. " +
			"Defaults to the current month.",
	}, s.monthlyReport)

	addTool(s, mcpServer, &mcp.Tool{
		Name:  "get_range_report",
		Title: "Get range report",
		Description: "Income, spending and savings over a range of months (from/to as YYYY-MM, default the " +
			"current year to date), with monthly averages, best and worst month, and a category breakdown " +
			"with per-month amounts. Budgets are scaled by the months counted. The range is clamped to start " +
			"no earlier than the user's first transaction and end no later than the current month; from/to " +
			"in the response show the months actually counted. For month-by-month figures use " +
			"get_spending_trend.",
	}, s.rangeReport)

	addTool(s, mcpServer, &mcp.Tool{
		Name:  "get_budget_status",
		Title: "Get budget status",
		Description: "Budget usage for every category with a monthly budget, for one month (default current). " +
			"Status is under (<80%), near (80-100%) or over (>100%). Sorted by percentage used, highest first.",
	}, s.budgetStatus)

	addTool(s, mcpServer, &mcp.Tool{
		Name:  "get_spending_trend",
		Title: "Get spending trend",
		Description: "Monthly income, spending and savings for the last N months (default 12, max 36), " +
			"oldest first, ending with the current month.",
	}, s.spendingTrend)

	addTool(s, mcpServer, &mcp.Tool{
		Name:  "list_uncategorized",
		Title: "List uncategorized expenses",
		Description: "Transactions with no category, grouped by identical description and sorted by count. " +
			"Useful for suggesting new category patterns.",
	}, s.listUncategorized)
}

// --- list_expenses ---------------------------------------------------------

type listExpensesInput struct {
	Description string   `json:"description,omitempty" jsonschema:"case-insensitive substring match on the description"`
	Source      string   `json:"source,omitempty"      jsonschema:"case-insensitive substring match on the source (bank or card)"`
	AmountMin   *float64 `json:"amount_min,omitempty"  jsonschema:"minimum signed amount in major units; charges are negative"`
	AmountMax   *float64 `json:"amount_max,omitempty"  jsonschema:"maximum signed amount in major units; charges are negative"`
	DateFrom    string   `json:"date_from,omitempty"   jsonschema:"earliest date, inclusive, YYYY-MM-DD"`
	DateTo      string   `json:"date_to,omitempty"     jsonschema:"latest date, inclusive, YYYY-MM-DD"`
	Type        string   `json:"type,omitempty"        jsonschema:"only return charge or income transactions"`
	Category    string   `json:"category,omitempty"    jsonschema:"exact category name; omit to return expenses from all categories"`
	Sort        string   `json:"sort,omitempty"        jsonschema:"one of date:desc (default), date:asc, amount:asc, amount:desc"`
	Limit       int      `json:"limit,omitempty"       jsonschema:"maximum rows to return, default 50, max 500"`
}

type expenseOutput struct {
	ID          int64   `json:"id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Source      string  `json:"source"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Type        string  `json:"type"`
	Category    string  `json:"category"    jsonschema:"an empty category means the expense does not belong to any category"`
}

type listExpensesOutput struct {
	Expenses      []expenseOutput `json:"expenses"`
	Returned      int             `json:"returned"`
	TotalMatching int             `json:"total_matching"`
	Truncated     bool            `json:"truncated"`
}

func (s *server) listExpenses(ctx context.Context, userID int64, in listExpensesInput) (listExpensesOutput, error) {
	filter, sortOpts, err := buildFilter(in)
	if err != nil {
		return listExpensesOutput{}, err
	}

	limit, err := clampLimit(in.Limit, defaultExpenseLimit, maxExpenseLimit)
	if err != nil {
		return listExpensesOutput{}, err
	}

	if in.Type != "" && in.Type != typeCharge && in.Type != typeIncome {
		return listExpensesOutput{}, invalidArg("type must be charge or income")
	}

	categoryNames, err := s.categoryNames(ctx, userID)
	if err != nil {
		return listExpensesOutput{}, err
	}

	if in.Category != "" && !containsValue(categoryNames, in.Category) {
		return listExpensesOutput{}, invalidArg(
			fmt.Sprintf("unknown category %q; call list_categories for valid names", in.Category),
		)
	}

	expenses, err := s.Expense.List(ctx, userID, filter, sortOpts)
	if err != nil {
		return listExpensesOutput{}, err
	}

	out := listExpensesOutput{Expenses: []expenseOutput{}}

	for _, ex := range expenses {
		row := toExpenseOutput(ex, categoryNames)
		if in.Type != "" && row.Type != in.Type {
			continue
		}
		if !matchesCategory(row.Category, in.Category) {
			continue
		}

		out.TotalMatching++

		if len(out.Expenses) < limit {
			out.Expenses = append(out.Expenses, row)
		}
	}

	out.Returned = len(out.Expenses)
	out.Truncated = out.TotalMatching > out.Returned

	return out, nil
}

func matchesCategory(rowCategory, want string) bool {
	switch want {
	case "":
		return true
	default:
		return rowCategory == want
	}
}

func buildFilter(in listExpensesInput) (*domain.ExpenseFilter, *domain.SortOptions, error) {
	filter := &domain.ExpenseFilter{}

	if in.Description != "" {
		filter.Description = &in.Description
	}
	if in.Source != "" {
		filter.Source = &in.Source
	}
	if in.AmountMin != nil {
		v := toCents(*in.AmountMin)
		filter.AmountMin = &v
	}
	if in.AmountMax != nil {
		v := toCents(*in.AmountMax)
		filter.AmountMax = &v
	}
	if in.DateFrom != "" {
		from, err := parseDate(in.DateFrom)
		if err != nil {
			return nil, nil, invalidArg("date_from must be YYYY-MM-DD")
		}
		filter.DateFrom = &from
	}
	if in.DateTo != "" {
		to, err := parseDate(in.DateTo)
		if err != nil {
			return nil, nil, invalidArg("date_to must be YYYY-MM-DD")
		}
		// Make the bound cover the whole day.
		endOfDay := to.AddDate(0, 0, 1).Add(-time.Second)
		filter.DateTo = &endOfDay
	}

	sortOpts := domain.DefaultSortOptions()
	if in.Sort != "" {
		parsed, ok := parseSort(in.Sort)
		if !ok {
			return nil, nil, invalidArg("sort must be one of date:desc, date:asc, amount:asc, amount:desc")
		}
		sortOpts = parsed
	}

	return filter, sortOpts, nil
}

func parseSort(s string) (*domain.SortOptions, bool) {
	switch s {
	case "date:desc":
		return &domain.SortOptions{Field: domain.SortByDate, Direction: domain.SortDesc}, true
	case "date:asc":
		return &domain.SortOptions{Field: domain.SortByDate, Direction: domain.SortAsc}, true
	case "amount:desc":
		return &domain.SortOptions{Field: domain.SortByAmount, Direction: domain.SortDesc}, true
	case "amount:asc":
		return &domain.SortOptions{Field: domain.SortByAmount, Direction: domain.SortAsc}, true
	default:
		return nil, false
	}
}

func toExpenseOutput(ex domain.Expense, categoryNames map[int64]string) expenseOutput {
	row := expenseOutput{
		ID:          ex.ID(),
		Date:        ex.Date().Format(dateLayout),
		Description: ex.Description(),
		Source:      ex.Source(),
		Amount:      toUnits(ex.Amount()),
		Currency:    ex.Currency(),
		Type:        expenseTypeName(ex.Type()),
	}

	if id := ex.CategoryID(); id != nil {
		row.Category = categoryNames[*id]
	}

	return row
}

// --- list_categories -------------------------------------------------------

type listCategoriesInput struct{}

type categoryOutput struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Pattern       string  `json:"pattern"        jsonschema:"regex matched against expense descriptions"`
	MonthlyBudget float64 `json:"monthly_budget" jsonschema:"monthly budget in major units, 0 means no budget"`
	Excluded      bool    `json:"excluded"       jsonschema:"true for the category whose expenses are left out of reports"`
}

type listCategoriesOutput struct {
	Categories []categoryOutput `json:"categories"`
}

func (s *server) listCategories(
	ctx context.Context,
	userID int64,
	_ listCategoriesInput,
) (listCategoriesOutput, error) {
	categories, err := s.Category.List(ctx, userID)
	if err != nil {
		return listCategoriesOutput{}, err
	}

	out := listCategoriesOutput{Categories: make([]categoryOutput, 0, len(categories))}
	for _, c := range categories {
		out.Categories = append(out.Categories, categoryOutput{
			ID:            c.ID(),
			Name:          c.Name(),
			Pattern:       c.Pattern(),
			MonthlyBudget: toUnits(c.MonthlyBudget()),
			Excluded:      c.Name() == domain.ExcludeCategory,
		})
	}

	return out, nil
}

// --- get_monthly_report ----------------------------------------------------

type monthInput struct {
	Month int `json:"month,omitempty" jsonschema:"month number 1-12, defaults to the current month"`
	Year  int `json:"year,omitempty"  jsonschema:"four-digit year, defaults to the current year"`
}

type budgetOutput struct {
	Budget         float64 `json:"budget"`
	Spent          float64 `json:"spent"`
	Remaining      float64 `json:"remaining"       jsonschema:"negative when over budget"`
	PercentageUsed float64 `json:"percentage_used"`
	Status         string  `json:"status"          jsonschema:"under, near or over"`
}

type categorySummaryOutput struct {
	Name               string        `json:"name"`
	Amount             float64       `json:"amount"                         jsonschema:"total for the period as a positive amount"`
	PercentageOfTotal  float64       `json:"percentage_of_total"            jsonschema:"share of total spending (expense categories) or total income (income categories)"`
	PercentageOfIncome float64       `json:"percentage_of_income,omitempty" jsonschema:"expense categories only: share of total income; can exceed 100; omitted when there was no income"`
	Transactions       int           `json:"transactions"`
	AverageAmount      float64       `json:"average_amount"`
	LastTransaction    string        `json:"last_transaction"`
	Budget             *budgetOutput `json:"budget,omitempty"               jsonschema:"present only when the category has a budget"`
}

type monthlyReportOutput struct {
	Month                 string                  `json:"month"                    jsonschema:"YYYY-MM"`
	Title                 string                  `json:"title"`
	Income                float64                 `json:"income"`
	Spending              float64                 `json:"spending"                 jsonschema:"total charges as a positive amount"`
	Savings               float64                 `json:"savings"                  jsonschema:"income minus spending"`
	SavingsPercentage     float64                 `json:"savings_percentage"`
	AverageSpendingPerDay float64                 `json:"average_spending_per_day"`
	EarningsPerDay        float64                 `json:"earnings_per_day"`
	ExpenseCategories     []categorySummaryOutput `json:"expense_categories"`
	IncomeCategories      []categorySummaryOutput `json:"income_categories"`
}

func (s *server) monthlyReport(ctx context.Context, userID int64, in monthInput) (monthlyReportOutput, error) {
	month, year, err := s.resolveMonth(in)
	if err != nil {
		return monthlyReportOutput{}, err
	}

	rep, err := s.Report.Month(ctx, userID, month, year)
	if err != nil {
		return monthlyReportOutput{}, err
	}

	return monthlyReportOutput{
		Month:                 monthKey(year, month),
		Title:                 rep.Title,
		Income:                toUnits(rep.Income),
		Spending:              toUnits(abs(rep.Spending)),
		Savings:               toUnits(rep.Savings),
		SavingsPercentage:     round2(float64(rep.SavingsPercentage)),
		AverageSpendingPerDay: toUnits(rep.AverageSpendingPerDay),
		EarningsPerDay:        toUnits(rep.EarningsPerDay),
		ExpenseCategories:     toCategorySummaries(rep.ExpenseCategories),
		IncomeCategories:      toCategorySummaries(rep.IncomeCategories),
	}, nil
}

func toCategorySummaries(categories []domain.CategoryReport) []categorySummaryOutput {
	out := make([]categorySummaryOutput, 0, len(categories))
	for _, c := range categories {
		summary := categorySummaryOutput{
			Name:               c.Name,
			Amount:             toUnits(abs(c.Amount)),
			PercentageOfTotal:  round2(c.PercentageOfTotal),
			PercentageOfIncome: round2(c.PercentageOfIncome),
			Transactions:       len(c.Expenses),
			AverageAmount:      toUnits(abs(c.AvgAmount)),
		}
		if !c.LastTransaction.IsZero() {
			summary.LastTransaction = c.LastTransaction.Format(dateLayout)
		}
		if c.Budget.Status != domain.BudgetStatusNoBudget && c.Budget.Amount > 0 {
			b := toBudgetOutput(c.Budget)
			summary.Budget = &b
		}
		out = append(out, summary)
	}
	return out
}

func toBudgetOutput(b domain.BudgetInfo) budgetOutput {
	return budgetOutput{
		Budget:         toUnits(b.Amount),
		Spent:          toUnits(b.Spent),
		Remaining:      toUnits(b.Remaining),
		PercentageUsed: round2(b.PercentageUsed),
		Status:         string(b.Status),
	}
}

// --- get_range_report ------------------------------------------------------

type rangeInput struct {
	From string `json:"from,omitempty" jsonschema:"first month YYYY-MM, defaults to January of the current year"`
	To   string `json:"to,omitempty"   jsonschema:"last month YYYY-MM, defaults to the current month"`
}

type monthHighlightOutput struct {
	Month             string  `json:"month"              jsonschema:"YYYY-MM"`
	Savings           float64 `json:"savings"`
	SavingsPercentage float64 `json:"savings_percentage"`
}

type rangeCategoryOutput struct {
	categorySummaryOutput
	PerMonth float64 `json:"per_month" jsonschema:"amount divided by months_counted"`
}

type rangeReportOutput struct {
	From                   string                `json:"from"                     jsonschema:"first month counted, YYYY-MM; may be later than requested"`
	To                     string                `json:"to"                       jsonschema:"last month counted, YYYY-MM; may be earlier than requested"`
	Title                  string                `json:"title"`
	MonthsCounted          int                   `json:"months_counted"`
	Income                 float64               `json:"income"`
	Spending               float64               `json:"spending"                 jsonschema:"total charges as a positive amount"`
	Savings                float64               `json:"savings"                  jsonschema:"income minus spending"`
	SavingsPercentage      float64               `json:"savings_percentage"`
	AverageMonthlyIncome   float64               `json:"average_monthly_income"`
	AverageMonthlySpending float64               `json:"average_monthly_spending" jsonschema:"positive amount"`
	BestMonth              *monthHighlightOutput `json:"best_month,omitempty"     jsonschema:"month with the highest savings; omitted when fewer than two months have data"`
	WorstMonth             *monthHighlightOutput `json:"worst_month,omitempty"    jsonschema:"month with the lowest savings; omitted when fewer than two months have data"`
	ExpenseCategories      []rangeCategoryOutput `json:"expense_categories"`
	IncomeCategories       []rangeCategoryOutput `json:"income_categories"`
}

func (s *server) rangeReport(ctx context.Context, userID int64, in rangeInput) (rangeReportOutput, error) {
	now := s.now()
	from := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	var err error
	if in.From != "" {
		if from, err = time.ParseInLocation(monthLayout, in.From, time.Local); err != nil {
			return rangeReportOutput{}, invalidArg("from must be a month in YYYY-MM format")
		}
	}
	if in.To != "" {
		if to, err = time.ParseInLocation(monthLayout, in.To, time.Local); err != nil {
			return rangeReportOutput{}, invalidArg("to must be a month in YYYY-MM format")
		}
	}

	sel, err := s.Report.Range(ctx, userID, from, to)
	if err != nil {
		return rangeReportOutput{}, err
	}

	out := rangeReportOutput{
		Title:                  sel.Title,
		MonthsCounted:          sel.MonthCount,
		Income:                 toUnits(sel.Income),
		Spending:               toUnits(abs(sel.Spending)),
		Savings:                toUnits(sel.Savings),
		SavingsPercentage:      round2(float64(sel.SavingsPercentage)),
		AverageMonthlyIncome:   toUnits(sel.PerMonth(sel.Income)),
		AverageMonthlySpending: toUnits(abs(sel.PerMonth(sel.Spending))),
		BestMonth:              toMonthHighlight(sel.Best),
		WorstMonth:             toMonthHighlight(sel.Worst),
		ExpenseCategories:      toRangeCategories(sel.Report, sel.ExpenseCategories),
		IncomeCategories:       toRangeCategories(sel.Report, sel.IncomeCategories),
	}
	if n := len(sel.Months); n > 0 {
		out.From = monthKey(sel.Months[0].Year, sel.Months[0].Month)
		out.To = monthKey(sel.Months[n-1].Year, sel.Months[n-1].Month)
	}

	return out, nil
}

func toMonthHighlight(m *domain.MonthSummary) *monthHighlightOutput {
	if m == nil {
		return nil
	}
	return &monthHighlightOutput{
		Month:             monthKey(m.Year, m.Month),
		Savings:           toUnits(m.Savings),
		SavingsPercentage: round2(float64(m.SavingsPercentage)),
	}
}

func toRangeCategories(rep domain.Report, categories []domain.CategoryReport) []rangeCategoryOutput {
	summaries := toCategorySummaries(categories)
	out := make([]rangeCategoryOutput, 0, len(summaries))
	for i, summary := range summaries {
		out = append(out, rangeCategoryOutput{
			categorySummaryOutput: summary,
			PerMonth:              toUnits(abs(rep.PerMonth(categories[i].Amount))),
		})
	}
	return out
}

// --- get_budget_status -----------------------------------------------------

type categoryBudgetOutput struct {
	Category string `json:"category"`
	budgetOutput
}

type budgetStatusOutput struct {
	Month      string                 `json:"month"      jsonschema:"YYYY-MM"`
	Categories []categoryBudgetOutput `json:"categories"`
}

func (s *server) budgetStatus(ctx context.Context, userID int64, in monthInput) (budgetStatusOutput, error) {
	month, year, err := s.resolveMonth(in)
	if err != nil {
		return budgetStatusOutput{}, err
	}

	budgets, err := s.Report.Budgets(ctx, userID, month, year)
	if err != nil {
		return budgetStatusOutput{}, err
	}

	out := budgetStatusOutput{
		Month:      monthKey(year, month),
		Categories: make([]categoryBudgetOutput, 0, len(budgets)),
	}
	for _, b := range budgets {
		out.Categories = append(out.Categories, categoryBudgetOutput{
			Category:     b.Name,
			budgetOutput: toBudgetOutput(b.Budget),
		})
	}

	return out, nil
}

// --- get_spending_trend ----------------------------------------------------

type spendingTrendInput struct {
	Months int `json:"months,omitempty" jsonschema:"number of months to include, default 12, max 36"`
}

type trendPointOutput struct {
	Month             string  `json:"month"              jsonschema:"YYYY-MM"`
	Income            float64 `json:"income"`
	Spending          float64 `json:"spending"           jsonschema:"total charges as a positive amount"`
	Savings           float64 `json:"savings"`
	SavingsPercentage float64 `json:"savings_percentage"`
}

type spendingTrendOutput struct {
	Months []trendPointOutput `json:"months"`
}

func (s *server) spendingTrend(ctx context.Context, userID int64, in spendingTrendInput) (spendingTrendOutput, error) {
	n, err := clampLimit(in.Months, defaultTrendMonths, maxTrendMonths)
	if err != nil {
		return spendingTrendOutput{}, invalidArg("months must be between 1 and 36")
	}

	now := s.now()
	current := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	out := spendingTrendOutput{Months: make([]trendPointOutput, 0, n)}
	for i := n - 1; i >= 0; i-- {
		m := current.AddDate(0, -i, 0)

		rep, repErr := s.Report.Month(ctx, userID, int(m.Month()), m.Year())
		if repErr != nil {
			return spendingTrendOutput{}, repErr
		}

		out.Months = append(out.Months, trendPointOutput{
			Month:             monthKey(m.Year(), int(m.Month())),
			Income:            toUnits(rep.Income),
			Spending:          toUnits(abs(rep.Spending)),
			Savings:           toUnits(rep.Savings),
			SavingsPercentage: round2(float64(rep.SavingsPercentage)),
		})
	}

	return out, nil
}

// --- list_uncategorized ----------------------------------------------------

type listUncategorizedInput struct {
	Query string `json:"query,omitempty" jsonschema:"only include descriptions containing this text"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum groups to return, default 25, max 200"`
}

type uncategorizedGroupOutput struct {
	Description string  `json:"description"`
	Count       int     `json:"count"`
	Total       float64 `json:"total"       jsonschema:"signed sum; charges are negative"`
	LastDate    string  `json:"last_date"`
}

type listUncategorizedOutput struct {
	TotalExpenses int                        `json:"total_expenses"`
	TotalAmount   float64                    `json:"total_amount"`
	Groups        []uncategorizedGroupOutput `json:"groups"`
	Truncated     bool                       `json:"truncated"`
}

func (s *server) listUncategorized(
	ctx context.Context,
	userID int64,
	in listUncategorizedInput,
) (listUncategorizedOutput, error) {
	limit, err := clampLimit(in.Limit, defaultUncategorizedLimit, maxUncategorizedLimit)
	if err != nil {
		return listUncategorizedOutput{}, err
	}

	groups, keys, totalExpenses, totalAmount, err := s.Category.GetUncategorized(ctx, userID, in.Query)
	if err != nil {
		return listUncategorizedOutput{}, err
	}

	out := listUncategorizedOutput{
		TotalExpenses: totalExpenses,
		TotalAmount:   toUnits(totalAmount),
		Groups:        make([]uncategorizedGroupOutput, 0, min(limit, len(keys))),
		Truncated:     len(keys) > limit,
	}

	for _, key := range keys[:min(limit, len(keys))] {
		g := groups[key]
		group := uncategorizedGroupOutput{
			Description: key,
			Count:       g.Count,
			Total:       toUnits(g.Total),
		}
		// GetUncategorized sorts each group's expenses newest first.
		if len(g.Expenses) > 0 {
			group.LastDate = g.Expenses[0].Date().Format(dateLayout)
		}
		out.Groups = append(out.Groups, group)
	}

	return out, nil
}

// --- helpers ---------------------------------------------------------------

func (s *server) categoryNames(ctx context.Context, userID int64) (map[int64]string, error) {
	categories, err := s.Category.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	names := make(map[int64]string, len(categories))
	for _, c := range categories {
		names[c.ID()] = c.Name()
	}

	return names, nil
}

func (s *server) resolveMonth(in monthInput) (int, int, error) {
	now := s.now()
	month, year := int(now.Month()), now.Year()

	if in.Month != 0 {
		if in.Month < 1 || in.Month > 12 {
			return 0, 0, invalidArg("month must be between 1 and 12")
		}
		month = in.Month
	}

	if in.Year != 0 {
		if in.Year < 1900 || in.Year > 9999 {
			return 0, 0, invalidArg("year must be a four-digit year")
		}
		year = in.Year
	}

	return month, year, nil
}

func clampLimit(value, def, maximum int) (int, error) {
	switch {
	case value == 0:
		return def, nil
	case value < 0:
		return 0, invalidArg("limit must be positive")
	case value > maximum:
		return maximum, nil
	default:
		return value, nil
	}
}

func parseDate(s string) (time.Time, error) {
	return time.Parse(dateLayout, s)
}

func monthKey(year, month int) string {
	return fmt.Sprintf("%04d-%02d", year, month)
}

func expenseTypeName(t domain.ExpenseType) string {
	if t == domain.IncomeType {
		return typeIncome
	}
	return typeCharge
}

func containsValue(m map[int64]string, want string) bool {
	for _, v := range m {
		if v == want {
			return true
		}
	}
	return false
}

func toUnits(cents int64) float64 {
	return float64(cents) / centsPerUnit
}

func toCents(units float64) int64 {
	return int64(math.Round(units * centsPerUnit))
}

func round2(f float64) float64 {
	return math.Round(f*centsPerUnit) / centsPerUnit
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
