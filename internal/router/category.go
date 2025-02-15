package router

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"golang.org/x/exp/maps"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

func categoriesHandler(db *sql.DB, w http.ResponseWriter) {
	categoriesWithTotalExpenses, err := expenseDB.GetCategoriesAndTotalExpenses(db)
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

	err = categoriesTempl.Execute(w, data)
	if err != nil {
		log.Print(err.Error())
		errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
		w.Write([]byte(errorMessage))
	}
}

type reportExpense struct {
	Count   int
	Dates   []time.Time
	Amounts []int64
}

func uncategorizedHandler(db *sql.DB, matcher *category.Matcher, w http.ResponseWriter) {
	expenses, err := expenseDB.GetExpensesWithoutCategory(db)
	if err != nil {
		data := struct {
			Error error
		}{
			Error: err,
		}
		err = uncategoriesTempl.ExecuteTemplate(w, "uncategorized.html", data)
		if err != nil {
			log.Print(err.Error())
			errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
			w.Write([]byte(errorMessage))
			return
		}
	}

	groupedExpenses := map[string]reportExpense{}

	for _, ex := range expenses {
		if r, ok := groupedExpenses[ex.Description]; ok {
			r.Count++
			r.Dates = append(r.Dates, ex.Date)
			r.Amounts = append(r.Amounts, ex.AmountWithSign())
			groupedExpenses[ex.Description] = r
		} else {
			groupedExpenses[ex.Description] = reportExpense{
				Count: 1,
				Dates: []time.Time{
					ex.Date,
				},
				Amounts: []int64{
					ex.AmountWithSign(),
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
		Categories:      matcher.Categories(),
		Error:           nil,
	}
	err = uncategoriesTempl.ExecuteTemplate(w, "uncategorized.html", data)
	if err != nil {
		log.Print(err.Error())
		errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
		w.Write([]byte(errorMessage))
		return
	}
}

func updateCategoryHandler(db *sql.DB, matcher *category.Matcher, w http.ResponseWriter, r *http.Request) {
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
		err = uncategoriesTempl.ExecuteTemplate(w, "uncategorized.html", data)
		if err != nil {
			log.Print(err.Error())
			errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
			w.Write([]byte(errorMessage))
			return
		}
	}

	expenses, err := expenseDB.SearchExpensesByDescription(db, expenseDescription)

	if err != nil {
		log.Println("error SearchExpensesByDescription ", err.Error())

		data := struct {
			Error error
		}{
			Error: err,
		}
		err = uncategoriesTempl.ExecuteTemplate(w, "uncategorized.html", data)
		if err != nil {
			log.Print(err.Error())
			errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
			w.Write([]byte(errorMessage))
			return
		}
	}

	if len(expenses) > 0 {
		for _, ex := range expenses {
			ex.CategoryID = categoryID
		}

		updated, err := expenseDB.UpdateExpenses(db, expenses)
		if err != nil {
			log.Println("error UpdateExpenses ", err.Error())

			data := struct {
				Error error
			}{
				Error: err,
			}
			err = uncategoriesTempl.ExecuteTemplate(w, "uncategorized.html", data)
			if err != nil {
				log.Print(err.Error())
				errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
				w.Write([]byte(errorMessage))
				return
			}
		}

		if updated != int64(len(expenses)) {
			log.Print("not all expenses updated succesfully")
		}

		uncategorizedHandler(db, matcher, w)
	}
}
