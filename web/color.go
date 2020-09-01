package web

import (
	"fmt"
	"html/template"

	"github.com/lucasb-eyer/go-colorful"
)

// This file implements the color gradient stuff
// It's overkill, but i want to make it right

/* Most of this GradientTable stuff is directly copied from https://github.com/lucasb-eyer/go-colorful/blob/master/doc/gradientgen/gradientgen.go, credit to the author */

type gTable []struct {
	Col colorful.Color
	Pos float64
}

func (g gTable) Interpolate(t float64) colorful.Color {
	for i := 0; i < len(g)-1; i++ {
		c1 := g[i]
		c2 := g[i+1]
		if c1.Pos <= t && t <= c2.Pos {
			// We are in between c1 and c2. Go blend them!
			t := (t - c1.Pos) / (c2.Pos - c1.Pos)
			return c1.Col.BlendLab(c2.Col, t).Clamped()
		}
	}

	return g[len(g)-1].Col
}

func mustParseHex(s string) colorful.Color {
	c, err := colorful.Hex(s)
	if err != nil {
		panic("MustParseHex: " + err.Error())
	}
	return c
}

func gradient(score, maxscore int, table gTable) template.CSS {
	if score < 0 {
		score = 0
	}
	if score > maxscore {
		score = maxscore
	}
	percent := float64(score) / float64(maxscore)

	color := table.Interpolate(percent)

	// When it's perfect, we want to have another color for all this
	if percent == 1.0 {
		color = mustParseHex("#81f542")
	}

	return template.CSS(fmt.Sprint("background-color:", color.Hex()))
}
