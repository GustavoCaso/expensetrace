package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/service/apitoken"
	"github.com/GustavoCaso/expensetrace/service/category"
	"github.com/GustavoCaso/expensetrace/service/expense"
	"github.com/GustavoCaso/expensetrace/service/report"
	"github.com/GustavoCaso/expensetrace/storage"
	"github.com/GustavoCaso/expensetrace/testutil"
)

type fixture struct {
	storage storage.Storage
	user    domain.User
	tokens  *apitoken.Service
	server  *httptest.Server
}

func setup(t *testing.T) *fixture {
	t.Helper()

	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	tokens := apitoken.New(s, logger)

	handler := NewHandler(Services{
		Expense:  expense.New(s, logger),
		Category: category.New(s, logger),
		Report:   report.New(s, logger),
		Tokens:   tokens,
	}, logger)

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return &fixture{storage: s, user: user, tokens: tokens, server: srv}
}

func (f *fixture) token(t *testing.T, user domain.User) string {
	t.Helper()
	_, plaintext, validationErr, err := f.tokens.Create(t.Context(), user.ID(), "test")
	if err != nil || validationErr != nil {
		t.Fatalf("create token: %v %v", err, validationErr)
	}
	return plaintext
}

type bearerTransport struct {
	token string
}

func (b bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(req)
}

func (f *fixture) connect(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:   f.server.URL,
		HTTPClient: &http.Client{Transport: bearerTransport{token: token}},
		MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	return session
}

func call[Out any](t *testing.T, session *mcp.ClientSession, name string, args map[string]any) Out {
	t.Helper()

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("CallTool %s returned error: %s", name, toolText(res))
	}

	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var out Out
	if err = json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s output: %v (%s)", name, err, raw)
	}
	return out
}

func toolText(res *mcp.CallToolResult) string {
	var parts []string
	for _, c := range res.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, " ")
}

func (f *fixture) seed(t *testing.T, user domain.User) time.Time {
	t.Helper()
	ctx := t.Context()

	groceriesID, err := f.storage.CreateCategory(ctx, user.ID(), "Groceries", "market", 10000)
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), 1, 12, 0, 0, 0, time.Local)

	expenses := []domain.Expense{
		domain.NewExpense(0, "visa", "Super market", "EUR", -4550, today, domain.ChargeType, &groceriesID),
		domain.NewExpense(0, "visa", "Night market", "EUR", -4000, today, domain.ChargeType, &groceriesID),
		domain.NewExpense(0, "visa", "Coffee shop", "EUR", -350, today, domain.ChargeType, nil),
		domain.NewExpense(0, "bank", "Salary", "EUR", 300000, today, domain.IncomeType, nil),
	}
	if _, err = f.storage.InsertExpenses(ctx, user.ID(), expenses); err != nil {
		t.Fatalf("insert expenses: %v", err)
	}

	return today
}

func TestRejectsMissingAndInvalidTokens(t *testing.T) {
	f := setup(t)

	for name, header := range map[string]string{
		"missing":   "",
		"invalid":   "Bearer et_nope",
		"malformed": "Basic abc",
	} {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, f.server.URL,
				strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			if header != "" {
				req.Header.Set("Authorization", header)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", resp.StatusCode)
			}
		})
	}
}

func TestListToolsAreReadOnly(t *testing.T) {
	f := setup(t)
	session := f.connect(t, f.token(t, f.user))

	res, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	names := []string{}
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %s should be annotated read-only", tool.Name)
		}
	}

	for _, want := range []string{
		"list_expenses", "list_categories", "get_monthly_report",
		"get_budget_status", "get_spending_trend", "list_uncategorized",
	} {
		if !slices.Contains(names, want) {
			t.Errorf("missing tool %s (have %v)", want, names)
		}
	}
}

func TestListExpenses(t *testing.T) {
	f := setup(t)
	f.seed(t, f.user)
	session := f.connect(t, f.token(t, f.user))

	all := call[listExpensesOutput](t, session, "list_expenses", nil)
	if all.TotalMatching != 4 || all.Returned != 4 || all.Truncated {
		t.Fatalf("unexpected totals: %+v", all)
	}

	charges := call[listExpensesOutput](t, session, "list_expenses", map[string]any{
		"type": "charge",
		"sort": "amount:asc",
	})
	if charges.TotalMatching != 3 {
		t.Fatalf("expected 3 charges, got %d", charges.TotalMatching)
	}
	if first := charges.Expenses[0]; first.Amount != -45.50 || first.Category != "Groceries" ||
		first.Currency != "EUR" {
		t.Errorf("unexpected first charge: %+v", first)
	}

	limited := call[listExpensesOutput](t, session, "list_expenses", map[string]any{"limit": 1})
	if limited.Returned != 1 || !limited.Truncated || limited.TotalMatching != 4 {
		t.Errorf("expected truncated single row: %+v", limited)
	}

	big := call[listExpensesOutput](t, session, "list_expenses", map[string]any{"amount_max": -40})
	if big.TotalMatching != 2 {
		t.Errorf("expected 2 charges of 40 or more, got %+v", big)
	}
}

func TestListExpensesInvalidArguments(t *testing.T) {
	f := setup(t)
	session := f.connect(t, f.token(t, f.user))

	for name, args := range map[string]map[string]any{
		"bad type":     {"type": "refund"},
		"bad sort":     {"sort": "name:asc"},
		"bad date":     {"date_from": "01/02/2026"},
		"bad category": {"category": "Nope"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "list_expenses", Arguments: args})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			if !res.IsError {
				t.Errorf("expected tool error for %v", args)
			}
			if text := toolText(res); strings.Contains(text, "internal") {
				t.Errorf("validation errors should be specific, got %q", text)
			}
		})
	}
}

func TestMonthlyReportAndBudgets(t *testing.T) {
	f := setup(t)
	today := f.seed(t, f.user)

	// A budgeted category with no spending this month must still show up.
	if _, err := f.storage.CreateCategory(t.Context(), f.user.ID(), "Travel", "flight", 50000); err != nil {
		t.Fatal(err)
	}

	session := f.connect(t, f.token(t, f.user))
	args := map[string]any{"month": int(today.Month()), "year": today.Year()}

	rep := call[monthlyReportOutput](t, session, "get_monthly_report", args)
	if rep.Income != 3000 || rep.Spending != 89 || rep.Savings != 2911 {
		t.Errorf("unexpected totals: income=%v spending=%v savings=%v", rep.Income, rep.Spending, rep.Savings)
	}

	var groceries *categorySummaryOutput
	for i := range rep.ExpenseCategories {
		if rep.ExpenseCategories[i].Name == "Groceries" {
			groceries = &rep.ExpenseCategories[i]
		}
	}
	if groceries == nil {
		t.Fatalf("Groceries missing from report: %+v", rep.ExpenseCategories)
	}
	if groceries.Amount != 85.50 || groceries.Transactions != 2 || groceries.Budget == nil ||
		groceries.Budget.Status != string(domain.BudgetStatusNear) {
		t.Errorf("unexpected groceries summary: %+v budget=%+v", groceries, groceries.Budget)
	}

	budgets := call[budgetStatusOutput](t, session, "get_budget_status", args)
	if len(budgets.Categories) != 2 {
		t.Fatalf("expected 2 budgeted categories, got %+v", budgets.Categories)
	}
	if budgets.Categories[0].Category != "Groceries" || budgets.Categories[0].PercentageUsed != 85.5 {
		t.Errorf("expected Groceries first at 85.5%%, got %+v", budgets.Categories[0])
	}
	if travel := budgets.Categories[1]; travel.Category != "Travel" || travel.Spent != 0 ||
		travel.Status != string(domain.BudgetStatusUnder) {
		t.Errorf("unexpected travel budget: %+v", travel)
	}
}

func TestSpendingTrendAndUncategorized(t *testing.T) {
	f := setup(t)
	f.seed(t, f.user)
	session := f.connect(t, f.token(t, f.user))

	trend := call[spendingTrendOutput](t, session, "get_spending_trend", map[string]any{"months": 3})
	if len(trend.Months) != 3 {
		t.Fatalf("expected 3 months, got %d", len(trend.Months))
	}
	if last := trend.Months[2]; last.Spending != 89 || last.Income != 3000 {
		t.Errorf("current month should hold seeded data: %+v", last)
	}
	if trend.Months[0].Month >= trend.Months[2].Month {
		t.Errorf("trend should be oldest first: %+v", trend.Months)
	}

	uncategorized := call[listUncategorizedOutput](t, session, "list_uncategorized", nil)
	// Only charges count as uncategorized; the uncategorized salary is income.
	if uncategorized.TotalExpenses != 1 || len(uncategorized.Groups) != 1 ||
		uncategorized.Groups[0].Description != "Coffee shop" {
		t.Errorf("unexpected uncategorized: %+v", uncategorized)
	}

	categories := call[listCategoriesOutput](t, session, "list_categories", nil)
	var sawGroceries bool
	for _, c := range categories.Categories {
		if c.Name == "Groceries" && c.MonthlyBudget == 100 && c.Pattern == "market" {
			sawGroceries = true
		}
	}
	if !sawGroceries {
		t.Errorf("Groceries category missing or wrong: %+v", categories.Categories)
	}
}

func TestDataIsIsolatedPerToken(t *testing.T) {
	f := setup(t)
	f.seed(t, f.user)

	other, err := f.storage.CreateUser(t.Context(), "other", "hash")
	if err != nil {
		t.Fatal(err)
	}

	session := f.connect(t, f.token(t, other))

	expenses := call[listExpensesOutput](t, session, "list_expenses", nil)
	if expenses.TotalMatching != 0 {
		t.Errorf("other user should see no expenses, got %d", expenses.TotalMatching)
	}

	categories := call[listCategoriesOutput](t, session, "list_categories", nil)
	// Every user gets their own Exclude category on creation; nothing else.
	for _, c := range categories.Categories {
		if !c.Excluded {
			t.Errorf("other user should not see category %+v", c)
		}
	}
}
