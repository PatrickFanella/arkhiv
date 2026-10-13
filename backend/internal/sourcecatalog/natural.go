package sourcecatalog

import (
	"regexp"
	"strings"
)

var naturalParts = regexp.MustCompile(`[0-9]+|[^0-9]+`)

func naturalLess(a, b string) bool {
	left, right := naturalParts.FindAllString(a, -1), naturalParts.FindAllString(b, -1)
	for i := 0; i < len(left) && i < len(right); i++ {
		x, y := left[i], right[i]
		if x == y {
			continue
		}
		numeric := x[0] >= '0' && x[0] <= '9' && y[0] >= '0' && y[0] <= '9'
		if numeric {
			xx, yy := strings.TrimLeft(x, "0"), strings.TrimLeft(y, "0")
			if len(xx) != len(yy) {
				return len(xx) < len(yy)
			}
			if xx != yy {
				return xx < yy
			}
			continue
		}
		return x < y
	}
	if len(left) != len(right) {
		return len(left) < len(right)
	}
	return a < b
}
