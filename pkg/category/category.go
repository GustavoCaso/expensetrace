package category

import (
	"regexp"
)

var Exclude = "exclude"

type matcher struct {
	re       *regexp.Regexp
	category string
}

type Category struct {
	matchers []matcher
}

func New(categories map[string]string) Category {
	matchers := []matcher{}

	for key, matchString := range categories {
		m := matcher{
			re:       regexp.MustCompile(matchString),
			category: key,
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
