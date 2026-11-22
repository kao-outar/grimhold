package dice

import (
	"math/rand"
	"testing"
)

func TestParseExpr(t *testing.T) {
	tests := []struct {
		in            string
		n, sides, bon int
	}{
		{"1d6", 1, 6, 0},
		{"2d8+3", 2, 8, 3},
		{"1d4-1", 1, 4, -1},
		{"d20", 1, 20, 0},
		{"5", 0, 0, 5},
	}
	for _, tt := range tests {
		n, sides, bon, err := ParseExpr(tt.in)
		if err != nil {
			t.Fatalf("ParseExpr(%q): %v", tt.in, err)
		}
		if n != tt.n || sides != tt.sides || bon != tt.bon {
			t.Fatalf("ParseExpr(%q)=%d,%d,%d want %d,%d,%d", tt.in, n, sides, bon, tt.n, tt.sides, tt.bon)
		}
	}
}

func TestRollExprSeeded(t *testing.T) {
	r := Rand{R: rand.New(rand.NewSource(42))}
	total, err := RollExpr(r, "2d6+1")
	if err != nil {
		t.Fatal(err)
	}
	if total < 3 || total > 13 {
		t.Fatalf("total %d out of range", total)
	}

	r2 := Rand{R: rand.New(rand.NewSource(42))}
	total2, _ := RollExpr(r2, "2d6+1")
	if total != total2 {
		t.Fatalf("seeded rolls differ: %d vs %d", total, total2)
	}
}
