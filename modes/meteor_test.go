package modes

import "testing"

func TestRadiantMeteorsPointBackAtRadiant(t *testing.T) {
	st := &State{Width: 80, Height: 24}
	st.City = NewSkyline(3, 80, 24)
	st.RadiantX, st.RadiantY = 40, 2
	st.Meteors = make([]Meteor, 8)
	for i := 0; i < 40; i++ {
		st.spawnRadiantMeteor()
	}
	checked := 0
	for i := range st.Meteors {
		m := &st.Meteors[i]
		if !m.Active {
			continue
		}
		// each meteor must move away from the radiant over time, so its
		// trail points back at the shower origin
		d0 := (m.X-st.RadiantX)*(m.X-st.RadiantX) + (m.Y-st.RadiantY)*(m.Y-st.RadiantY)
		p2x := m.X + m.VX*5
		p2y := m.Y + m.VY*5
		d1 := (p2x-st.RadiantX)*(p2x-st.RadiantX) + (p2y-st.RadiantY)*(p2y-st.RadiantY)
		if d1 <= d0 {
			t.Fatalf("meteor not moving away from radiant: pos (%v,%v) vel (%v,%v)", m.X, m.Y, m.VX, m.VY)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no meteors spawned")
	}
}

func TestMeteorVisibilityGating(t *testing.T) {
	night := &State{Night: 1, WeatherLive: true}
	night.Weather.Condition = CondClear
	clear := night.meteorVisibility()
	night.Weather.Condition = CondCloudy
	cloudy := night.meteorVisibility()
	if cloudy >= clear {
		t.Fatalf("overcast should cut visibility: clear %v cloudy %v", clear, cloudy)
	}
	day := &State{Night: 0, WeatherLive: true}
	day.Weather.Condition = CondClear
	if day.meteorVisibility() >= clear {
		t.Fatal("night should boost the rate over day")
	}
}

func TestBolideTintsSky(t *testing.T) {
	st := &State{Width: 40, Height: 24}
	st.City = NewSkyline(3, 40, 24)
	st.Meteors = make([]Meteor, 4)
	st.BolideFlash = 0
	st.BolideChance = 1.0
	st.spawnFireball()
	if st.BolideFlash <= 0 {
		t.Fatal("bolide should set a sky tint")
	}
}
