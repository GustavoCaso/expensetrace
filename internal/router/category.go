package router

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

// enhancedCategory extends db.Category with extra UI-friendly fields.
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
	uncategorizedInfos, err := expenseDB.GetExpensesWithoutCategory(router.db)
	if err != nil {
		categoryIndexError(router, w, err)
		return
	}
	uncategorizedCount := len(uncategorizedInfos)

	// Get total categorized count
	totalCategorized := 0

	// Enhance categories with additional data
	enhancedCategories := make([]enhancedCategory, len(categories))

	for i, cat := range categories {
		// Get expenses for this category
		expenses, expensesErr := expenseDB.GetExpensesByCategory(router.db, cat.ID)

		if expensesErr != nil {
			categoryIndexError(router, w, expensesErr)
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

	categoryEntry, err := expenseDB.GetCategory(router.db, int64(categoryID))

	if err != nil {
		categoryIndexError(router, w, err)
		return
	}

	// Get expenses for this category
	expenses, err := expenseDB.GetExpensesByCategory(router.db, categoryEntry.ID)

	if err != nil {
		categoryIndexError(router, w, err)
		return
	}

	enhancedCat := createEnhancedCategory(categoryEntry, expenses)

	//nolint:nestif // No need to extract this code to a function for now as is clear
	if (pattern != "" && categoryEntry.Pattern != pattern) ||
		(name != "" && categoryEntry.Name != name) {
		patternChanged := false
		updated := false

		if pattern != "" && categoryEntry.Pattern != pattern {
			_, err = regexp.Compile(pattern)

			if err != nil {
				enhancedCat.Errors = true
				enhancedCat.ErrorStrings = map[string]string{
					"pattern": fmt.Sprintf("invalid pattern %v", err),
				}

				router.templates.Render(w, "partials/categories/card.html", enhancedCat)
				return
			}
			patternChanged = true
		}

		if name != "" && categoryEntry.Name != name || patternChanged {
			err = expenseDB.UpdateCategory(router.db, categoryID, name, pattern)

			if err != nil {
				enhancedCat.Errors = true
				enhancedCat.ErrorStrings = map[string]string{
					"name": fmt.Sprintf("failed to updated category %v", err),
				}

				router.templates.Render(w, "partials/categories/card.html", enhancedCat)
				return
			}

			updated = true
		}

		if updated {
			categories, categoryErr := expenseDB.GetCategories(router.db)
			if err != nil {
				categoryIndexError(router, w, categoryErr)
				return
			}

			matcher := category.NewMatcher(categories)
			router.matcher = matcher

			if patternChanged {
				allExpenses, expensesErr := expenseDB.GetExpenses(router.db)

				if expensesErr != nil {
					categoryIndexError(router, w, expensesErr)
					return
				}

				toUpdated := []*expenseDB.Expense{}

				for _, ex := range allExpenses {
					id, _ := matcher.Match(ex.Description)
					if id.Valid {
						if ex.CategoryID.Valid {
							// Update exiting expense with a new category
							if id.Int64 != ex.CategoryID.Int64 {
								ex.CategoryID = id
								toUpdated = append(toUpdated, ex)
							}
							// Update exiting expense without category with category
						} else {
							ex.CategoryID = id
							toUpdated = append(toUpdated, ex)
						}
					} else {
						if ex.CategoryID.Valid {
							// Changing a category pattern could render exiting expenses to have category NULL
							ex.CategoryID = sql.NullInt64{}
							toUpdated = append(toUpdated, ex)
						}
					}
				}

				if len(toUpdated) > 0 {
					updated, updateErr := expenseDB.UpdateExpenses(router.db, toUpdated)
					if updateErr != nil {
						categoryIndexError(router, w, updateErr)
						return
					}

					if int(updated) != len(toUpdated) {
						fmt.Println("not all categories were updated")
					}
				}
			}
		}

		enhancedCat.Category.Name = name
		enhancedCat.Category.Pattern = pattern

		if updated {
			router.resetCache()
		}
	}

	router.templates.Render(w, "partials/categories/card.html", enhancedCat)
}

type uncategorizedInfo struct {
	Count    int
	Expenses []struct {
		Date   time.Time
		Amount int64
	}
	Total int64
	Slug  string
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

	uncategorizeInfo := map[string]uncategorizedInfo{}
	totalExpenses := 0
	var totalAmount int64

	for _, ex := range expenses {
		if r, ok := uncategorizeInfo[ex.Description]; ok {
			r.Count++
			r.Expenses = append(r.Expenses, struct {
				Date   time.Time
				Amount int64
			}{
				Date:   ex.Date,
				Amount: ex.Amount,
			})
			r.Total += ex.Amount
			uncategorizeInfo[ex.Description] = r
		} else {
			uncategorizeInfo[ex.Description] = uncategorizedInfo{
				Count: 1,
				Total: ex.Amount,
				Expenses: []struct {
					Date   time.Time
					Amount int64
				}{
					{
						Date:   ex.Date,
						Amount: ex.Amount,
					},
				},
				Slug: slugify(ex.Description),
			}
		}

		totalExpenses++
		totalAmount += ex.Amount
	}

	keys := slices.Collect(maps.Keys(uncategorizeInfo))

	sort.SliceStable(keys, func(i, j int) bool {
		return uncategorizeInfo[keys[i]].Count > uncategorizeInfo[keys[j]].Count
	})

	for _, report := range uncategorizeInfo {
		sort.SliceStable(report.Expenses, func(i, j int) bool {
			return report.Expenses[i].Date.After(report.Expenses[j].Date)
		})
	}

	data := struct {
		Keys              []string
		UncategorizeInfo  map[string]uncategorizedInfo
		ExpenseCategories []expenseDB.Category
		IncomeCategories  []expenseDB.Category
		TotalExpenses     int
		TotalAmount       int64
		Error             error
	}{
		Keys:              keys,
		UncategorizeInfo:  uncategorizeInfo,
		ExpenseCategories: router.matcher.ExpenseCategories(),
		IncomeCategories:  router.matcher.IncomeCategories(),
		TotalExpenses:     totalExpenses,
		TotalAmount:       totalAmount,
		Error:             nil,
	}
	router.templates.Render(w, "pages/categories/uncategorized.html", data)
}

var specialCharactersRegex = regexp.MustCompile(`[^a-z0-9\-]`)
var multipleHyphenRegex = regexp.MustCompile(`[^a-z0-9\-]`)

func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)
	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	// Remove special characters
	s = specialCharactersRegex.ReplaceAllString(s, "")
	// Replace multiple hyphens with a single one
	s = multipleHyphenRegex.ReplaceAllString(s, "-")
	// Remove leading and trailing hyphens
	s = strings.Trim(s, "-")
	return s
}

func (router *router) updateUncategorizedHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("error r.ParseForm() ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

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

		updated, updateErr := expenseDB.UpdateExpenses(router.db, expenses)
		if updateErr != nil {
			data := struct {
				Error error
			}{
				Error: updateErr,
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
	err := r.ParseForm()
	if err != nil {
		log.Println("error r.ParseForm() ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		router.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	name := r.FormValue("name")
	pattern := r.FormValue("pattern")

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
	}

	if name == "" || pattern == "" {
		data.Error = errors.New(
			"category must include name and a valid regex pattern. Ensure that you populate the name and pattern input",
		)

		router.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	re, err := regexp.Compile(pattern)

	if err != nil {
		data.Error = err

		router.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	expenses, err := expenseDB.GetExpensesWithoutCategory(router.db)

	if err != nil {
		data.Error = err

		router.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	toUpdated := []*expenseDB.Expense{}

	for _, ex := range expenses {
		if re.MatchString(ex.Description) {
			toUpdated = append(toUpdated, ex)
		}
	}

	total := len(toUpdated)

	if create && total > 0 {
		categoryID, createErr := expenseDB.CreateCategory(router.db, name, pattern)

		if createErr != nil {
			data.Error = createErr

			router.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		sqlCategoryID := sql.NullInt64{Int64: categoryID, Valid: true}

		for _, ex := range expenses {
			ex.CategoryID = sqlCategoryID
		}

		updated, updateErr := expenseDB.UpdateExpenses(router.db, toUpdated)
		if updateErr != nil {
			data.Error = updateErr

			router.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		if int(updated) != total {
			fmt.Println("not all categories were updated")

			total = int(updated)
		}

		categories, categoryErr := expenseDB.GetCategories(router.db)
		if categoryErr != nil {
			data.Error = categoryErr

			router.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		matcher := category.NewMatcher(categories)
		router.matcher = matcher
		router.resetCache()
	}

	data.Total = total
	data.Results = toUpdated
	data.Create = create

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
