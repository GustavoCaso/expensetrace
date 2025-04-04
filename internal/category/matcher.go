package category

import (
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

func (c Matcher) Match(s string) (*int, string) {
	for _, matcher := range c.matchers {
		if matcher.re.MatchString(s) {
			return &matcher.id, matcher.category
		}
	}

	return nil, ""
}
