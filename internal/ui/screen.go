package ui

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	escClear   = "\033[2J\033[H"
	escReset   = "\033[0m"
	escBold    = "\033[1m"
	escDim     = "\033[2m"
	escRed     = "\033[31m"
	escGreen   = "\033[32m"
	escYellow  = "\033[33m"
	escCyan    = "\033[36m"
	escMagenta = "\033[35m"
)

func ClearHome(w io.Writer) {
	fmt.Fprint(w, escClear)
}

func HPBar(cur, max, width int) string {
	if max <= 0 {
		max = 1
	}
	if cur < 0 {
		cur = 0
	}
	if width < 4 {
		width = 10
	}
	filled := cur * width / max
	if cur > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	color := escGreen
	pct := cur * 100 / max
	if pct <= 25 {
		color = escRed
	} else if pct <= 50 {
		color = escYellow
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("%s[%s]%s %d/%d", color, bar, escReset, cur, max)
}

func DangerLabel(danger int) string {
	color := escGreen
	switch {
	case danger >= 4:
		color = escBold + escRed
	case danger == 3:
		color = escRed
	case danger == 2:
		color = escMagenta
	case danger == 1:
		color = escYellow
	}
	return fmt.Sprintf("%sdanger %d%s", color, danger, escReset)
}

func pad(s string, width int) string {
	n := visibleLen(s)
	if n >= width {
		return trim(s, width)
	}
	return s + strings.Repeat(" ", width-n)
}

func trim(s string, width int) string {
	if visibleLen(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	var b strings.Builder
	vis := 0
	limit := width - 1
	for i := 0; i < len(s) && vis < limit; {
		if seq, n := ansiAt(s, i); n > 0 {
			b.WriteString(seq)
			i += n
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		b.WriteRune(r)
		i += size
		vis++
	}
	b.WriteString("…")
	b.WriteString(escReset)
	return b.String()
}

func visibleLen(s string) int {
	return utf8.RuneCountInString(stripANSI(s))
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if _, n := ansiAt(s, i); n > 0 {
			i += n
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func ansiAt(s string, i int) (string, int) {
	if i >= len(s) || s[i] != 0x1b || i+1 >= len(s) || s[i+1] != '[' {
		return "", 0
	}
	j := i + 2
	for j < len(s) {
		c := s[j]
		j++
		if c >= 0x40 && c <= 0x7E {
			return s[i:j], j - i
		}
	}
	return "", 0
}

func hline(width int, title string) string {
	if title == "" {
		return "├" + strings.Repeat("─", width-2) + "┤"
	}

	inner := width - 2
	t := "─ " + title + " "
	tn := utf8.RuneCountInString(t)
	if tn >= inner {
		return "├" + trim(t, inner) + "┤"
	}
	return "├" + t + strings.Repeat("─", inner-tn) + "┤"
}

func topLine(width int, title string) string {
	inner := width - 2
	t := "─ " + title + " "
	tn := utf8.RuneCountInString(t)
	if tn >= inner {
		return "┌" + trim(t, inner) + "┐"
	}
	return "┌" + t + strings.Repeat("─", inner-tn) + "┐"
}

func bottomLine(width int) string {
	return "└" + strings.Repeat("─", width-2) + "┘"
}

func row(width int, content string) string {
	inner := width - 4
	return "│ " + pad(content, inner) + " │"
}
