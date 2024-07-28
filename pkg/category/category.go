package category

import "regexp"

type CategoryMatcher struct {
	re       *regexp.Regexp
	category string
}

var shoppingMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`amazon|amzn|books`),
	category: "shopping",
}
var payrollMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`datadog cloud spain`),
	category: "payroll",
}
var finesMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`dgt sanciones`),
	category: "fines",
}
var bizunChargeMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`cargo bizum`),
	category: "bizum charge",
}
var bizunIncomeMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`abono bizum`),
	category: "bizum income",
}

var subscriptionMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`youtubepremium`),
	category: "subscriptions",
}

var restarantsMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`bar|cerveza|ginos`),
	category: "restaurants",
}

var groceriesMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`supermercado|lidl|ahorramas|syra coffee`),
	category: "groceries",
}

var homeSpendingsMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`ferreteria|leroy merlin|ikea|el corte ingles venta|fronda`),
	category: "home",
}

var gasMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`ballenoil`),
	category: "gas",
}

var sportEquipmentMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`decathlon`),
	category: "sport equipment",
}

var sportMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`sputnik climbing`),
	category: "sport",
}

var entertaimentMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`nintendo`),
	category: "entertaiment",
}

var rentIncomeMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`trf\. grace cuddy`),
	category: "rent income",
}

var communityChargeMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`rcbo\.c\.p. ercilla`),
	category: "community spending",
}

var insurancesMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`rcbo\.santa lucia`),
	category: "insurances",
}

var lightMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`rcbo\.iberdrola`),
	category: "light",
}

var mortgageMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`rcbo\. préstamo`),
	category: "mortgage",
}

var irpfMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`trf\. devoluciones tributarias`),
	category: "irpf",
}

var interestMatcher = CategoryMatcher{
	re:       regexp.MustCompile(`ints\.plazo`),
	category: "interest",
}

var allMatchers = []CategoryMatcher{
	shoppingMatcher,
	payrollMatcher,
	finesMatcher,
	bizunChargeMatcher,
	bizunIncomeMatcher,
	subscriptionMatcher,
	restarantsMatcher,
	groceriesMatcher,
	homeSpendingsMatcher,
	gasMatcher,
	sportEquipmentMatcher,
	sportMatcher,
	entertaimentMatcher,
	rentIncomeMatcher,
	communityChargeMatcher,
	insurancesMatcher,
	lightMatcher,
	mortgageMatcher,
	irpfMatcher,
	interestMatcher,
}

func Match(s string) string {
	for _, matcher := range allMatchers {
		if matcher.re.MatchString(s) {
			return matcher.category
		}
	}

	return ""
}
