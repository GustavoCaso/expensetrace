package server

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

func (s *server) categoriesHandler(w http.ResponseWriter) {
	categories, err := expenseDB.GetCategories(s.db)

	if err != nil {
		categoryIndexError(s, w, fmt.Errorf("error fetch categories: %v", err.Error()))
		return
	}

	// Get counts for uncategorized expenses
	uncategorizedInfos, err := expenseDB.GetExpensesWithoutCategory(s.db)
	if err != nil {
		categoryIndexError(s, w, err)
		return
	}
	uncategorizedCount := len(uncategorizedInfos)

	// Get total categorized count
	totalCategorized := 0

	// Enhance categories with additional data
	enhancedCategories := make([]enhancedCategory, len(categories))

	for i, cat := range categories {
		// Get expenses for this category
		expenses, expensesErr := expenseDB.GetExpensesByCategory(s.db, cat.ID)

		if expensesErr != nil {
			categoryIndexError(s, w, expensesErr)
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

	s.templates.Render(w, "pages/categories/index.html", data)
}

func (s *server) updateCategoryHandler(
	id, name, pattern string,
	categoryType expenseDB.CategoryType,
	w http.ResponseWriter,
) {
	categoryID, err := strconv.Atoi(id)

	if err != nil {
		categoryIndexError(s, w, err)
		return
	}

	categoryEntry, err := expenseDB.GetCategory(s.db, int64(categoryID))

	if err != nil {
		categoryIndexError(s, w, err)
		return
	}

	// Get expenses for this category
	expenses, err := expenseDB.GetExpensesByCategory(s.db, categoryEntry.ID)

	if err != nil {
		categoryIndexError(s, w, err)
		return
	}

	enhancedCat := createEnhancedCategory(categoryEntry, expenses)

	updated := false
	patternChanged := false
	if (pattern != "" && categoryEntry.Pattern != pattern) ||
		(name != "" && categoryEntry.Name != name || categoryEntry.Type != categoryType) {
		if pattern != "" && categoryEntry.Pattern != pattern {
			_, err = regexp.Compile(pattern)

			if err != nil {
				enhancedCat.Errors = true
				enhancedCat.ErrorStrings = map[string]string{
					"pattern": fmt.Sprintf("invalid pattern %v", err),
				}

				s.templates.Render(w, "partials/categories/card.html", enhancedCat)
				return
			}
			patternChanged = true
		}

		err = expenseDB.UpdateCategory(s.db, categoryID, name, pattern, categoryType)

		if err != nil {
			enhancedCat.Errors = true
			enhancedCat.ErrorStrings = map[string]string{
				"name": fmt.Sprintf("failed to updated category %v", err),
			}

			s.templates.Render(w, "partials/categories/card.html", enhancedCat)
			return
		}

		updated = true
	}

	//nolint:nestif // No need to extract this code to a function for now as is clear
	if updated {
		categories, categoryErr := expenseDB.GetCategories(s.db)
		if err != nil {
			categoryIndexError(s, w, categoryErr)
			return
		}

		matcher := category.NewMatcher(categories)
		s.matcher = matcher

		if patternChanged {
			allExpenses, expensesErr := expenseDB.GetExpenses(s.db)

			if expensesErr != nil {
				categoryIndexError(s, w, expensesErr)
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
				updated, updateErr := expenseDB.UpdateExpenses(s.db, toUpdated)
				if updateErr != nil {
					categoryIndexError(s, w, updateErr)
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
	enhancedCat.Category.Type = categoryType

	if patternChanged {
		updateCategoryMatcherErr := s.updateCategoryMatcher()
		if updateCategoryMatcherErr != nil {
			categoryIndexError(s, w, updateCategoryMatcherErr)
			return
		}
	}

	if updated {
		s.resetCache()
	}

	s.templates.Render(w, "partials/categories/card.html", enhancedCat)
}

type uncategorizedInfo struct {
	Count    int
	Expenses []struct {
		Date   time.Time
		Amount int64
		Source string
	}
	Total int64
	Slug  string
}

func (s *server) uncategorizedHandler(w http.ResponseWriter) {
	expenses, err := expenseDB.GetExpensesWithoutCategory(s.db)
	if err != nil {
		data := struct {
			Error error
		}{
			Error: err,
		}
		s.templates.Render(w, "pages/categories/uncategorized.html", data)
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
				Source string
			}{
				Date:   ex.Date,
				Amount: ex.Amount,
				Source: ex.Source,
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
					Source string
				}{
					{
						Date:   ex.Date,
						Amount: ex.Amount,
						Source: ex.Source,
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
		ExpenseCategories: s.matcher.ExpenseCategories(),
		IncomeCategories:  s.matcher.IncomeCategories(),
		TotalExpenses:     totalExpenses,
		TotalAmount:       totalAmount,
		Error:             nil,
	}
	s.templates.Render(w, "pages/categories/uncategorized.html", data)
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

func (s *server) updateUncategorizedHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("error r.ParseForm() ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		s.templates.Render(w, "pages/categories/uncategorized.html", data)
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
		s.templates.Render(w, "pages/categories/uncategorized.html", data)
	}

	cat, err := expenseDB.GetCategory(s.db, int64(categoryID))

	if err != nil {
		log.Println("error GetCategory ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		s.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	extendedRegex, err := extendRegex(cat.Pattern, expenseDescription)

	if err != nil {
		log.Println("error extendRegex ", err.Error())
		data := struct {
			Error error
		}{
			Error: err,
		}
		s.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	err = expenseDB.UpdateCategory(s.db, cat.ID, cat.Name, extendedRegex, cat.Type)
	if err != nil {
		log.Println("error UpdateCategory ", err.Error())
		data := struct {
			Error error
		}{
			Error: err,
		}
		s.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	expenses, err := expenseDB.SearchExpensesByDescription(s.db, expenseDescription)

	if err != nil {
		log.Println("error SearchExpensesByDescription ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		s.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	if len(expenses) > 0 {
		for _, ex := range expenses {
			ex.CategoryID = sql.NullInt64{Int64: int64(categoryID), Valid: true}
		}

		updated, updateErr := expenseDB.UpdateExpenses(s.db, expenses)
		if updateErr != nil {
			data := struct {
				Error error
			}{
				Error: updateErr,
			}
			s.templates.Render(w, "pages/categories/uncategorized.html", data)
			return
		}

		if updated != int64(len(expenses)) {
			log.Print("not all expenses updated succesfully")
		}

		updateCategoryMatcherErr := s.updateCategoryMatcher()
		if updateCategoryMatcherErr != nil {
			data := struct {
				Error error
			}{
				Error: updateCategoryMatcherErr,
			}
			s.templates.Render(w, "pages/categories/uncategorized.html", data)
			return
		}

		s.resetCache()
	}

	s.uncategorizedHandler(w)
}

func extendRegex(pattern, description string) (string, error) {
	extendedPattern := fmt.Sprintf("%s|%s", pattern, regexp.QuoteMeta(description))
	re, err := regexp.Compile(extendedPattern)
	if err != nil {
		return "", err
	}
	return re.String(), nil
}

func (s *server) createCategoryHandler(create bool, w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("error r.ParseForm() ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		s.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	name := r.FormValue("name")
	pattern := r.FormValue("pattern")
	categoryTypeStr := r.FormValue("type")
	categoryType := expenseDB.ExpenseCategoryType // Default to expense
	if categoryTypeStr == "1" {
		categoryType = expenseDB.IncomeCategoryType
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
	}

	if name == "" || pattern == "" {
		data.Error = errors.New(
			"category must include name and a valid regex pattern. Ensure that you populate the name and pattern input",
		)

		s.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	re, err := regexp.Compile(pattern)

	if err != nil {
		data.Error = err

		s.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	expenses, err := expenseDB.GetExpensesWithoutCategory(s.db)

	if err != nil {
		data.Error = err

		s.templates.Render(w, "partials/categories/new_result.html", data)
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
		categoryID, createErr := expenseDB.CreateCategory(s.db, name, pattern, categoryType)

		if createErr != nil {
			data.Error = createErr

			s.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		sqlCategoryID := sql.NullInt64{Int64: categoryID, Valid: true}

		for _, ex := range expenses {
			ex.CategoryID = sqlCategoryID
		}

		updated, updateErr := expenseDB.UpdateExpenses(s.db, toUpdated)
		if updateErr != nil {
			data.Error = updateErr

			s.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		if int(updated) != total {
			fmt.Println("not all categories were updated")

			total = int(updated)
		}

		updateCategoryMatcherErr := s.updateCategoryMatcher()
		if updateCategoryMatcherErr != nil {
			data.Error = updateCategoryMatcherErr
			s.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		s.resetCache()
	}

	data.Total = total
	data.Results = toUpdated
	data.Create = create

	s.templates.Render(w, "partials/categories/new_result.html", data)
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

func categoryIndexError(server *server, w io.Writer, err error) {
	data := struct {
		Error error
	}{
		Error: err,
	}
	server.templates.Render(w, "pages/categories/index.html", data)
}

func (s *server) updateCategoryMatcher() error {
	categories, categoryErr := expenseDB.GetCategories(s.db)
	if categoryErr != nil {
		return categoryErr
	}

	matcher := category.NewMatcher(categories)
	s.matcher = matcher
	return nil
}
