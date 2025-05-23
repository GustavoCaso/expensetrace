package category

import (
	"database/sql"
	"regexp"

	"github.com/GustavoCaso/expensetrace/internal/db"
)

var Exclude = "exclude"

type matcher struct {
	re       *regexp.Regexp
	category string
	id       int
}

type Matcher struct {
	matchers   []matcher
	categories []db.Category
}

func NewMatcher(categories []db.Category) *Matcher {
	matchers := make([]matcher, len(categories))

	for i, category := range categories {
		m := matcher{
			re:       regexp.MustCompile(category.Pattern),
			category: category.Name,
			id:       category.ID,
		}
		matchers[i] = m
	}

	return &Matcher{
		matchers:   matchers,
		categories: categories,
	}
}

func (c Matcher) Categories() []db.Category {
	return c.categories
}

func (c Matcher) ExpenseCategories() []db.Category {
	return c.categoriesByType(db.ExpenseCategoryType)
}

func (c Matcher) IncomeCategories() []db.Category {
	return c.categoriesByType(db.IncomeCategoryType)
}

func (c Matcher) categoriesByType(categoryType db.CategoryType) []db.Category {
	result := []db.Category{}
	for _, cat := range c.categories {
		if cat.Type == categoryType {
			result = append(result, cat)
		}
	}
	return result
}

func (c Matcher) Match(s string) (sql.NullInt64, string) {
	for _, matcher := range c.matchers {
		if matcher.re.MatchString(s) {
			return sql.NullInt64{Int64: int64(matcher.id), Valid: true}, matcher.category
		}
	}

	return sql.NullInt64{Int64: 0, Valid: false}, ""
}
