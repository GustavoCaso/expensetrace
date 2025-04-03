package router

import (
	"fmt"
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

func (router *router) categoriesHandler(w http.ResponseWriter) {
	categoriesWithTotalExpenses, err := expenseDB.GetCategoriesAndTotalExpenses(router.db)
	var data interface{}
	if err != nil {
		log.Print(err.Error())
		data = struct {
			Error error
		}{
			Error: fmt.Errorf("error fetch categories: %v", err.Error()),
		}
	} else {
		data = struct {
			Categories []expenseDB.Category
			Error      error
		}{
			Categories: categoriesWithTotalExpenses,
			Error:      nil,
		}
	}

	router.templates.Render(w, "pages/categories/index.html", data)
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
		router.templates.Render(w, "partials/categories/uncategorized.html", data)
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
	router.templates.Render(w, "partials/categories/uncategorized.html", data)
}

func (router *router) updateCategoryHandler(w http.ResponseWriter, r *http.Request) {
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
		router.templates.Render(w, "partials/categories/uncategorized.html", data)
	}

	_, err = expenseDB.GetCategory(router.db, categoryID)

	if err != nil {
		log.Println("error GetCategory ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		router.templates.Render(w, "partials/categories/uncategorized.html", data)
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
		router.templates.Render(w, "partials/categories/uncategorized.html", data)
		return
	}

	if len(expenses) > 0 {
		for _, ex := range expenses {
			ex.CategoryID = categoryID
		}

		updated, err := expenseDB.UpdateExpenses(router.db, expenses)
		if err != nil {
			data := struct {
				Error error
			}{
				Error: err,
			}
			router.templates.Render(w, "partials/categories/uncategorized.html", data)
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

		for _, ex := range expenses {
			ex.CategoryID = int(categoryID)
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
