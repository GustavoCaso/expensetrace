package category

import (
	"regexp"

	"github.com/GustavoCaso/expensetrace/internal/config"
)

var Exclude = "exclude"

type matcher struct {
	re       *regexp.Regexp
	category string
}

type Category struct {
	matchers []matcher
}

func New(categories []config.Category) Category {
	matchers := []matcher{}

	for _, category := range categories {
		m := matcher{
			re:       regexp.MustCompile(category.Pattern),
			category: category.Name,
		}
		matchers = append(matchers, m)
	}

	return Category{
		matchers: matchers,
	}
}

func (c Category) Match(s string) string {
	for _, matcher := range c.matchers {
		if matcher.re.MatchString(s) {
			return matcher.category
		}
	}

	return ""
}
