package modes

import (
	"math/rand"
	"time"

	"github.com/gdamore/tcell/v2"
)

type Building struct {
	X, W, H int
	Lit     []bool
}

type Skyline struct {
	Seed      int64
	Buildings []Building
	HorizonY  int
	Tallest   int
}

func NewSkyline(seed int64, w, h int) Skyline {
	sk := Skyline{Seed: seed, HorizonY: h * 2 / 3, Tallest: -1}
	if w <= 0 || h < 6 {
		return sk
	}
	rng := rand.New(rand.NewSource(seed))
	maxH := sk.HorizonY/2 + 1
	x := 0
	for x < w {
		bw := 5 + rng.Intn(9)
		if bw > w-x {
			bw = w - x
		}
		bh := 3 + rng.Intn(maxH)
		if bh > maxH {
			bh = maxH
		}
		lit := make([]bool, bw*bh)
		for i := range lit {
			lit[i] = rng.Float64() < 0.10
		}
		for i := bw * (bh - 2); i < len(lit); i++ {
			lit[i] = false
		}
		sk.Buildings = append(sk.Buildings, Building{X: x, W: bw, H: bh, Lit: lit})
		x += bw + 1
	}
	// force one building near the top so every city has a hero tower
	idx := rng.Intn(len(sk.Buildings))
	b := &sk.Buildings[idx]
	if b.H < maxH {
		old := b.Lit
		b.H = maxH
		b.Lit = make([]bool, b.W*b.H)
		copy(b.Lit, old)
	}
	for i := range sk.Buildings {
		if sk.Tallest < 0 || sk.Buildings[i].H > sk.Buildings[sk.Tallest].H {
			sk.Tallest = i
		}
	}
	return sk
}

func (st *State) InitWorld() {
	if st.Seed == 0 {
		st.Seed = time.Now().UnixNano()
	}
	st.City = NewSkyline(st.Seed, st.Width, st.Height)
	if st.World == WorldCoast {
		st.initCoast(st.Width, st.Height)
	}
	st.initClouds()
	st.updateRooftops()
	st.Night = 1 - daytime(timeHour())
}

// updateRooftops records the landing y per column. On the coast every column
// lands on the water at the horizon; the city lands on building roofs.
func (st *State) updateRooftops() {
	n := st.Width
	if n <= 0 {
		st.RoofTop = nil
		return
	}
	st.RoofTop = make([]int, n)
	for i := range st.RoofTop {
		st.RoofTop[i] = -1
	}
	if st.World == WorldCoast {
		for i := range st.RoofTop {
			st.RoofTop[i] = st.City.HorizonY
		}
		return
	}
	if len(st.City.Buildings) == 0 {
		return
	}
	for _, b := range st.City.Buildings {
		roof := st.City.HorizonY - 1 - b.H
		for x := b.X; x < b.X+b.W && x < n; x++ {
			if x >= 0 {
				st.RoofTop[x] = roof
			}
		}
	}
}

// DrawWorldSky renders everything behind the weather: sun, moon,
// stars, and clouds.
func DrawWorldSky(screen tcell.Screen, st *State) {
	if st.Width <= 0 || st.Height <= 0 {
		return
	}
	st.Night = 1 - daytime(timeHour())
	drawSkyGlow(screen, st)
	st.drawMoonStars(screen)
	st.drawSun(screen)
	st.drawClouds(screen)
}

// DrawCity renders the terrain in front of the weather: the coast's sea and
// lighthouse, or the city's skyline and ground.
func DrawCity(screen tcell.Screen, st *State) {
	if st.Width <= 0 || st.Height <= 0 {
		return
	}
	if st.World == WorldCoast {
		st.drawCoast(screen)
		return
	}
	drawSkyline(screen, st)
	drawGround(screen, st)
}

func blendHex(a, b uint32, t float64) tcell.Color {
	ar := float64((a >> 16) & 0xff)
	ag := float64((a >> 8) & 0xff)
	ab := float64(a & 0xff)
	br := float64((b >> 16) & 0xff)
	bg := float64((b >> 8) & 0xff)
	bb := float64(b & 0xff)
	r := int32(ar + (br-ar)*t)
	g := int32(ag + (bg-ag)*t)
	bl := int32(ab + (bb-ab)*t)
	return tcell.NewHexColor(r<<16 | g<<8 | bl)
}

// drawSkyGlow shows nothing on normal runs. The flat grey band appears only
// when live weather is offline, so raind never fakes weather without a
// visible tell.
func drawSkyGlow(screen tcell.Screen, st *State) {
	horizon := st.City.HorizonY
	if horizon < 4 || st.City.HorizonY == 0 {
		return
	}
	if !st.WeatherLive || !st.Weather.Offline {
		return
	}
	for i := 0; i < 4; i++ {
		y := horizon - 4 + i
		style := tcell.StyleDefault.Foreground(blendHex(0x22242c, 0x3c4048, float64(i)/3))
		for x := 0; x < st.Width; x++ {
			screen.SetContent(x, y, '▀', nil, style)
		}
	}
}

func drawSkyline(screen tcell.Screen, st *State) {
	horizon := st.City.HorizonY
	if horizon <= 0 || len(st.City.Buildings) == 0 {
		return
	}
	body := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x141821))
	roof := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x4d5870))
	litWin := tcell.StyleDefault.Foreground(tcell.NewHexColor(0xc9a26b)).Attributes(tcell.AttrDim)

	for _, b := range st.City.Buildings {
		for dy := 0; dy < b.H; dy++ {
			y := horizon - 1 - dy
			if y < 0 {
				break
			}
			for dx := 0; dx < b.W; dx++ {
				x := b.X + dx
				if x >= st.Width {
					break
				}
				if dy == 0 {
					screen.SetContent(x, y, '▀', nil, roof)
					continue
				}
				if b.Lit[dy*b.W+dx] && st.Night > 0.5 {
					screen.SetContent(x, y, '█', nil, litWin)
				} else {
					screen.SetContent(x, y, '█', nil, body)
				}
			}
		}
	}
}

func drawGround(screen tcell.Screen, st *State) {
	horizon := st.City.HorizonY
	if horizon <= 0 || horizon >= st.Height {
		return
	}
	top := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x151a22)).Background(tcell.NewHexColor(0x151a22))
	fill := tcell.StyleDefault.Foreground(tcell.NewHexColor(0x0a0e13)).Background(tcell.NewHexColor(0x0a0e13))
	for y := horizon; y < st.Height; y++ {
		style := fill
		if y == horizon {
			style = top
		}
		for x := 0; x < st.Width; x++ {
			screen.SetContent(x, y, '█', nil, style)
		}
	}
}
