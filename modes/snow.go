package modes

import (
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

// reinitializes snowflakes
func (st *State) InitSnow() {
	st.Wind = 0
	st.WindTarget = 0
	st.WindTick = 0
	st.initSnow()
}

// renders snow mode
func DrawSnow(screen tcell.Screen, st *State) {
	w, h := st.Width, st.Height
	if w == 0 || h == 0 {
		return
	}
	st.updateWind()
	mult := st.Speed.SpeedMultiplier()
	style := tcell.StyleDefault.Foreground(st.Color).Background(tcell.ColorBlack)

	lightChars := []rune{'·', '∗', '❄', '✻'}
	heavyChars := []rune{'|', '•'}

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
		if int(f.Y) >= h {
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
		if int(f.X) < 0 {
			f.X = float64(w - 1)
		}
		if int(f.X) >= w {
			f.X = 0
		}
	}
}
