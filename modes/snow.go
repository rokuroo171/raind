package modes

import (
	"math"
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

func (st *State) InitSnow() {
	st.Wind = 0
	st.WindTarget = 0
	st.WindTick = 0
	st.initSnow()
}

func DrawSnow(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	st.updateWind()
	mult := st.Speed.SpeedMultiplier()
	if st.FocusMode {
		mult *= 0.85
	}
	style := tcell.StyleDefault.Foreground(st.Color).Background(tcell.ColorReset)

	lightChars := []rune{'·', '∗', '❄', '✻'}
	heavyChars := []rune{'|', '•'}
	maxAccum := h / 6
	if maxAccum < 1 {
		maxAccum = 1
	}

	if len(st.AccumRow) != w {
		st.AccumRow = make([]int, w)
	}

	drawAuroraSky(screen, st)

	resetFlake := func(f *Snowflake) {
		f.Y = 0
		f.X = float64(rand.Intn(w))
		wgt := 0.2 + rand.Float64()*0.8
		f.Weight = wgt
		f.Speed = 0.08 + wgt*0.18
		f.Drift = (rand.Float64() - 0.5) * (0.25 - wgt*0.15)
		if wgt > 0.7 {
			f.Char = heavyChars[rand.Intn(len(heavyChars))]
		} else if wgt < 0.4 {
			f.Char = lightChars[rand.Intn(len(lightChars))]
		} else {
			all := append(append([]rune{}, lightChars...), heavyChars...)
			f.Char = all[rand.Intn(len(all))]
		}
	}

	depositSnow := func(col int) {
		if w == 0 {
			return
		}
		// On the coast snow fades into the sea; only the lighthouse rock
		// catches any, capped at a thin dusting.
		if st.World == WorldCoast {
			if col < st.Coast.LighthouseX-2 || col > st.Coast.LighthouseX+2 {
				return
			}
			if st.AccumRow[col] < 3 {
				st.AccumRow[col]++
			}
			return
		}
		best := col
		bestHeight := maxAccum + 1
		for dx := -1; dx <= 1; dx++ {
			c := col + dx
			if c < 0 || c >= w {
				continue
			}
			if st.AccumRow[c] < bestHeight {
				best = c
				bestHeight = st.AccumRow[c]
			}
		}
		if bestHeight < maxAccum {
			st.AccumRow[best]++
		}
	}
	smoothAccum := func() {
		if w < 2 {
			return
		}
		for pass := 0; pass < 2; pass++ {
			for x := 1; x < w; x++ {
				if st.AccumRow[x]-st.AccumRow[x-1] > 1 {
					st.AccumRow[x]--
					st.AccumRow[x-1]++
				}
			}
			for x := w - 2; x >= 0; x-- {
				if st.AccumRow[x]-st.AccumRow[x+1] > 1 {
					st.AccumRow[x]--
					st.AccumRow[x+1]++
				}
			}
		}
		// crest cleanup: soften single-cell spikes and fill tiny pits
		for x := 1; x < w-1; x++ {
			left := st.AccumRow[x-1]
			cur := st.AccumRow[x]
			right := st.AccumRow[x+1]
			if left == right {
				if cur > left+1 {
					st.AccumRow[x] = left + 1
				}
				if cur+1 < left {
					st.AccumRow[x] = left - 1
				}
			}
		}
	}

	for i := range st.Flakes {
		f := &st.Flakes[i]
		if !f.Active {
			continue
		}
		x := int(f.X)
		y := int(f.Y)
		if x >= 0 && x < w && y >= 0 && y < h {
			screen.SetContent(x, y, f.Char, nil, style)
		}
		f.Y += f.Speed * mult
		f.X += f.Drift*mult + st.Wind*(1.0-f.Weight)*0.3*mult
		if int(f.X) < 0 {
			f.X = float64(w - 1)
		}
		if int(f.X) >= w {
			f.X = 0
		}
		col := int(f.X)
		if col < 0 {
			col = 0
		}
		if col >= w {
			col = w - 1
		}
		accum := st.AccumRow[col]
		if accum > maxAccum {
			accum = maxAccum
			st.AccumRow[col] = accum
		}
		if int(f.Y) >= h {
			depositSnow(col)
			resetFlake(f)
			continue
		}
		if accum > 0 {
			topAccumY := h - accum
			if int(f.Y) >= topAccumY {
				depositSnow(col)
				resetFlake(f)
				continue
			}
		}
	}
	smoothAccum()
}

// DrawSnowForeground renders the accumulated snow and the pine tree on top
// of the city, since DrawSnow draws only the falling flakes behind it.
func DrawSnowForeground(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	accumStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	accumTopStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorReset)
	maxAccum := h / 6
	if maxAccum < 1 {
		maxAccum = 1
	}

	for x := 0; x < w; x++ {
		height := st.AccumRow[x]
		if height > maxAccum {
			height = maxAccum
			st.AccumRow[x] = height
		}
		for k := 0; k < height; k++ {
			y := h - 1 - k
			if y < 0 {
				break
			}
			ch := '·'
			style := accumStyle
			if k == height-1 {
				ch = '▁'
				style = accumTopStyle
			} else if k >= height-2 {
				ch = '▄'
			}
			screen.SetContent(x, y, ch, nil, style)
		}
	}
	if st.World == WorldCity {
		drawPine(screen, st)
	}
}

func drawAuroraSky(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w <= 0 || h <= 2 {
		return
	}
	bandTop := h / 3
	if bandTop < 3 {
		bandTop = 3
	}
	pulseBoost := 0.0
	if st.Frame%480 < 24 {
		pulseBoost = 0.18
	}
	if st.FocusMode {
		pulseBoost *= 0.5
	}
	for x := 0; x < w; x++ {
		wave := math.Sin(float64(x)*0.085+float64(st.Frame)*0.018)*0.55 +
			math.Sin(float64(x)*0.03-float64(st.Frame)*0.011)*0.45
		height := int((wave + 1.0) * 0.5 * float64(bandTop) * (0.82 + pulseBoost))
		if height < 2 {
			continue
		}
		for y := 0; y < height; y++ {
			intensity := 1.0 - float64(y)/float64(height)
			if intensity < 0.15 {
				continue
			}
			// sparse curtain: skip most mid-band cells for a softer glow
			skip := 4
			if intensity > 0.65 {
				skip = 3
			}
			if st.FocusMode {
				skip++
			}
			if (x+y+st.Frame)%skip != 0 {
				continue
			}
			phase := math.Sin(float64(x)*0.07 + float64(y)*0.12 + float64(st.Frame)*0.02)
			color := tcell.ColorTeal
			if phase > 0.35 {
				color = tcell.ColorGreen
			} else if phase < -0.2 {
				color = tcell.ColorPurple
			}
			style := tcell.StyleDefault.Foreground(color).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
			ch := '·'
			if intensity > 0.75 && (x+st.Frame)%5 == 0 {
				ch = '░'
			}
			screen.SetContent(x, y, ch, nil, style)
		}
	}
}

func drawPine(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w < 18 || h < 10 {
		return
	}
	tree := []string{
		"    ^    ",
		"   /|\\   ",
		"  //|\\\\  ",
		" ///|\\\\\\ ",
		"////|\\\\\\\\",
		"   |||   ",
		"   |||   ",
	}
	startX := w - len(tree[0]) - 2
	startY := h - len(tree) - 2
	leafStyleDim := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorReset).Attributes(tcell.AttrDim)
	leafStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorReset)
	trunkStyle := tcell.StyleDefault.Foreground(tcell.ColorMaroon).Background(tcell.ColorReset).Attributes(tcell.AttrDim)

	for y, row := range tree {
		for x, ch := range row {
			if ch == ' ' {
				continue
			}
			px := startX + x
			py := startY + y
			if px < 0 || px >= w || py < 0 || py >= h {
				continue
			}
			style := leafStyleDim
			if ch == '|' {
				style = trunkStyle
			} else if (px+py)%3 == 0 {
				style = leafStyle
			}
			screen.SetContent(px, py, ch, nil, style)
		}
	}
}
