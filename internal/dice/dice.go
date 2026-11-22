package dice

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

type Roller interface {
	Intn(n int) int
}

type Rand struct {
	R *rand.Rand
}

func (r Rand) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return r.R.Intn(n)
}

func D20(r Roller) int {
	return r.Intn(20) + 1
}

func Die(r Roller, sides int) int {
	if sides <= 0 {
		return 0
	}
	return r.Intn(sides) + 1
}

func ParseExpr(expr string) (n, sides, bonus int, err error) {
	expr = strings.TrimSpace(strings.ToLower(expr))
	if expr == "" {
		return 0, 0, 0, fmt.Errorf("empty dice expression")
	}
	if !strings.Contains(expr, "d") {
		v, e := strconv.Atoi(expr)
		if e != nil {
			return 0, 0, 0, fmt.Errorf("invalid dice %q: %w", expr, e)
		}
		return 0, 0, v, nil
	}
	parts := strings.SplitN(expr, "d", 2)
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("invalid dice %q", expr)
	}
	if parts[0] == "" {
		n = 1
	} else {
		n, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid dice count in %q: %w", expr, err)
		}
	}
	rest := parts[1]
	bonus = 0
	sidesStr := rest
	for i, c := range rest {
		if c == '+' || c == '-' {
			sidesStr = rest[:i]
			bonus, err = strconv.Atoi(rest[i:])
			if err != nil {
				return 0, 0, 0, fmt.Errorf("invalid dice bonus in %q: %w", expr, err)
			}
			break
		}
	}
	sides, err = strconv.Atoi(sidesStr)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid die size in %q: %w", expr, err)
	}
	if n < 0 || sides < 0 {
		return 0, 0, 0, fmt.Errorf("invalid dice %q", expr)
	}
	return n, sides, bonus, nil
}

func RollExpr(r Roller, expr string) (int, error) {
	n, sides, bonus, err := ParseExpr(expr)
	if err != nil {
		return 0, err
	}
	total := bonus
	for i := 0; i < n; i++ {
		total += Die(r, sides)
	}
	return total, nil
}

func Pick[T any](r Roller, items []T) T {
	var zero T
	if len(items) == 0 {
		return zero
	}
	return items[r.Intn(len(items))]
}
