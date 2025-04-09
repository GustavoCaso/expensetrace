package router

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"golang.org/x/exp/maps"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

// enhancedCategory extends db.Category with extra UI-friendly fields
type enhancedCategory struct {
	expenseDB.Category
	AvgAmount       int64
	LastTransaction string
	Total           int
	TotalAmount     int64
	SpendingCount   int
	IncomeCount     int
	Errors          bool
	ErrorStrings    map[string]string
}

func (router *router) categoriesHandler(w http.ResponseWriter) {
	categories, err := expenseDB.GetCategories(router.db)

	if err != nil {
		categoryIndexError(router, w, fmt.Errorf("error fetch categories: %v", err.Error()))
		return
	}

	// Get counts for uncategorized expenses
	uncategorizedExpenses, err := expenseDB.GetExpensesWithoutCategory(router.db)
	if err != nil {
		categoryIndexError(router, w, err)
		return
	}
	uncategorizedCount := len(uncategorizedExpenses)

	// Get total categorized count
	totalCategorized := 0

	// Enhance categories with additional data
	enhancedCategories := make([]enhancedCategory, len(categories))

	for i, cat := range categories {
		// Get expenses for this category
		expenses, err := expenseDB.GetExpensesByCategory(router.db, cat.ID)

		if err != nil {
			categoryIndexError(router, w, err)
			return
		}

		totalCategorized += len(expenses)

		enhancedCategories[i] = createEnhancedCategory(cat, expenses)
	}

	data := struct {
		Categories         []enhancedCategory
		CategorizedCount   int
		UncategorizedCount int
		Error              error
	}{
		Categories:         enhancedCategories,
		CategorizedCount:   totalCategorized,
		UncategorizedCount: uncategorizedCount,
		Error:              nil,
	}

	router.templates.Render(w, "pages/categories/index.html", data)
}

func (router *router) updateCategoryHandler(id, name, pattern string, w http.ResponseWriter) {
	categoryID, err := strconv.Atoi(id)

	if err != nil {
		categoryIndexError(router, w, err)
		return
	}

	category, err := expenseDB.GetCategory(router.db, int64(categoryID))

	if err != nil {
		categoryIndexError(router, w, err)
		return
	}

	// Get expenses for this category
	expenses, err := expenseDB.GetExpensesByCategory(router.db, category.ID)

	if err != nil {
		categoryIndexError(router, w, err)
		return
	}

	enhancedCategory := createEnhancedCategory(category, expenses)

	if category.Pattern != pattern || category.Name != name {
		_, err = regexp.Compile(pattern)

		if err != nil {
			enhancedCategory.Errors = true
			enhancedCategory.ErrorStrings = map[string]string{
				"pattern": fmt.Sprintf("invalid pattern %v", err),
			}

			router.templates.Render(w, "partials/categories/card.html", enhancedCategory)
			return
		}

		err = expenseDB.UpdateCategory(router.db, categoryID, name, pattern)

		if err != nil {
			enhancedCategory.Errors = true
			enhancedCategory.ErrorStrings = map[string]string{
				"name": fmt.Sprintf("not unique name %v", err),
			}

			router.templates.Render(w, "partials/categories/card.html", enhancedCategory)
			return
		}

		enhancedCategory.Category.Name = name
		enhancedCategory.Category.Pattern = pattern
	}

	router.templates.Render(w, "partials/categories/card.html", enhancedCategory)
}

type reportExpense struct {
	Count   int
	Dates   []time.Time
	Amounts []int64
}

func (router *router) uncategorizedHandler(w http.ResponseWriter) {
	expenses, err := expenseDB.GetExpensesWithoutCategory(router.db)
	if err != nil {
		data := struct {
			Error error
		}{
			Error: err,
		}
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	groupedExpenses := map[string]reportExpense{}

	for _, ex := range expenses {
		if r, ok := groupedExpenses[ex.Description]; ok {
			r.Count++
			r.Dates = append(r.Dates, ex.Date)
			r.Amounts = append(r.Amounts, ex.Amount)
			groupedExpenses[ex.Description] = r
		} else {
			groupedExpenses[ex.Description] = reportExpense{
				Count: 1,
				Dates: []time.Time{
					ex.Date,
				},
				Amounts: []int64{
					ex.Amount,
				},
			}
		}
	}

	keys := maps.Keys(groupedExpenses)

	sort.SliceStable(keys, func(i, j int) bool {
		return groupedExpenses[keys[i]].Count > groupedExpenses[keys[j]].Count
	})

	data := struct {
		Keys            []string
		GroupedExpenses map[string]reportExpense
		Categories      []expenseDB.Category
		Error           error
	}{
		Keys:            keys,
		GroupedExpenses: groupedExpenses,
		Categories:      router.matcher.Categories(),
		Error:           nil,
	}
	router.templates.Render(w, "pages/categories/uncategorized.html", data)
}

func (router *router) updateUncategorizedHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	expenseDescription := r.FormValue("description")
	categoryID, err := strconv.Atoi(r.FormValue("categoryID"))

	if err != nil {
		log.Println("error strconv.Atoi ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
	}

	_, err = expenseDB.GetCategory(router.db, int64(categoryID))

	if err != nil {
		log.Println("error GetCategory ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	expenses, err := expenseDB.SearchExpensesByDescription(router.db, expenseDescription)

	if err != nil {
		log.Println("error SearchExpensesByDescription ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	if len(expenses) > 0 {
		for _, ex := range expenses {
			ex.CategoryID = sql.NullInt64{Int64: int64(categoryID), Valid: true}
		}

		updated, err := expenseDB.UpdateExpenses(router.db, expenses)
		if err != nil {
			data := struct {
				Error error
			}{
				Error: err,
			}
			router.templates.Render(w, "pages/categories/uncategorized.html", data)
			return
		}

		if updated != int64(len(expenses)) {
			log.Print("not all expenses updated succesfully")
		}

		router.resetCache()
	}

	router.uncategorizedHandler(w)
}

func (router *router) createCategoryHandler(create bool, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	name := r.FormValue("name")
	pattern := r.FormValue("pattern")

	if name == "" || pattern == "" {
		data := struct {
			Error error
		}{
			Error: fmt.Errorf("category must include name and a valid regex pattern. Ensure that you populate the name and pattern input"),
		}

		router.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	re, err := regexp.Compile(pattern)

	if err != nil {
		data := struct {
			Error error
		}{
			Error: err,
		}

		router.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	expenses, _ := expenseDB.GetExpensesWithoutCategory(router.db)

	results := []*expenseDB.Expense{}

	for _, ex := range expenses {
		if re.MatchString(ex.Description) {
			results = append(results, ex)
		}
	}

	total := len(results)

	if create && total > 0 {
		categoryID, err := expenseDB.CreateCategory(router.db, name, pattern)

		if err != nil {
			data := struct {
				Error error
			}{
				Error: err,
			}

			router.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		sqlCategoryID := sql.NullInt64{Int64: int64(categoryID), Valid: true}

		for _, ex := range expenses {
			ex.CategoryID = sqlCategoryID
		}

		updated, err := expenseDB.UpdateExpenses(router.db, expenses)
		if err != nil {
			data := struct {
				Error error
			}{
				Error: err,
			}

			router.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		if int(updated) != total {
			fmt.Println("not all categories were updated")

			total = int(updated)
		}

		categories, err := expenseDB.GetCategories(router.db)
		if err != nil {
			data := struct {
				Error error
			}{
				Error: err,
			}

			router.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		matcher := category.NewMatcher(categories)
		router.matcher = matcher
		router.resetCache()
	}

	data := struct {
		Name    string
		Pattern string
		Results []*expenseDB.Expense
		Total   int
		Error   error
		Create  bool
	}{
		Name:    name,
		Pattern: pattern,
		Results: results,
		Total:   total,
		Error:   err,
		Create:  create,
	}

	router.templates.Render(w, "partials/categories/new_result.html", data)
}

func createEnhancedCategory(category expenseDB.Category, expenses []*expenseDB.Expense) enhancedCategory {
	// Calculate average amount and last transaction
	var totalAmount int64
	var lastTransaction time.Time
	spendingCount := 0
	incomeCount := 0

	for _, exp := range expenses {
		totalAmount += exp.Amount

		if exp.Amount < 0 {
			spendingCount++
		} else {
			incomeCount++
		}

		if lastTransaction.IsZero() || exp.Date.After(lastTransaction) {
			lastTransaction = exp.Date
		}
	}

	avgAmount := int64(0)
	if len(expenses) > 0 {
		avgAmount = totalAmount / int64(len(expenses))
	}

	lastTransactionStr := ""
	if !lastTransaction.IsZero() {
		lastTransactionStr = lastTransaction.Format("2006-01-02")
	}

	return enhancedCategory{
		Category:        category,
		AvgAmount:       avgAmount,
		LastTransaction: lastTransactionStr,
		Total:           len(expenses),
		TotalAmount:     totalAmount,
		SpendingCount:   spendingCount,
		IncomeCount:     incomeCount,
	}
}

func categoryIndexError(router *router, w io.Writer, err error) {
	data := struct {
		Error error
	}{
		Error: err,
	}
	router.templates.Render(w, "pages/categories/index.html", data)
}
