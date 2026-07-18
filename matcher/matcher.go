package matcher

import (
	"regexp"

	"github.com/GustavoCaso/expensetrace/domain"
)

var Exclude = "exclude"

type matcher struct {
	re       *regexp.Regexp
	category string
	id       int64
}

type Matcher struct {
	matchers   []matcher
	categories []domain.Category
}

func New(categories []domain.Category) *Matcher {
	matchers := make([]matcher, len(categories))

	for i, category := range categories {
		m := matcher{
			re:       regexp.MustCompile(category.Pattern()),
			category: category.Name(),
			id:       category.ID(),
		}
		matchers[i] = m
	}

	return &Matcher{
		matchers:   matchers,
		categories: categories,
	}
}

func (c Matcher) Categories() []domain.Category {
	return c.categories
}

func (c Matcher) Match(s string) (*int64, string) {
	for _, matcher := range c.matchers {
		if matcher.re.MatchString(s) {
			return &matcher.id, matcher.category
		}
	}

	return nil, ""
}
