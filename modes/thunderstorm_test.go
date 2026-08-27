package modes

import "testing"

func TestGenerateBoltStopsAtTarget(t *testing.T) {
	st := &State{Width: 40, Height: 24}
	st.City = NewSkyline(3, 40, 24)
	stop := st.City.HorizonY - 3
	bolt := st.generateBolt(40, 24, 3, 7, stop)
	if len(bolt.Points) == 0 {
		t.Fatal("expected bolt points")
	}
	last := bolt.Points[len(bolt.Points)-1]
	if last.Y != stop-1 {
		t.Fatalf("bolt did not stop at target: last Y %d, want %d", last.Y, stop-1)
	}
	for _, p := range bolt.Points {
		if p.Y < 0 || p.Y >= stop {
			t.Fatalf("bolt point out of range: %+v", p)
		}
	}
}

func TestRooftopsWithinBounds(t *testing.T) {
	st := &State{Width: 40, Height: 24, World: WorldCity}
	st.City = NewSkyline(5, 40, 24)
	st.updateRooftops()
	if len(st.RoofTop) != 40 {
		t.Fatalf("rooftop width %d, want 40", len(st.RoofTop))
	}
	found := false
	for x, roof := range st.RoofTop {
		if roof == -1 {
			continue
		}
		found = true
		if roof < 0 || roof >= st.City.HorizonY {
			t.Fatalf("roof at %d out of bounds: %d (horizon %d)", x, roof, st.City.HorizonY)
		}
	}
	if !found {
		t.Fatal("expected at least one roofed column")
	}
}

func TestCoastRooftopsLandOnWater(t *testing.T) {
	st := &State{Width: 40, Height: 24, World: WorldCoast}
	st.City = NewSkyline(5, 40, 24)
	st.initCoast(40, 24)
	st.updateRooftops()
	if len(st.RoofTop) != 40 {
		t.Fatalf("rooftop width %d, want 40", len(st.RoofTop))
	}
	for x, roof := range st.RoofTop {
		if roof != st.City.HorizonY {
			t.Fatalf("coast roof at %d = %d, want sea level %d", x, roof, st.City.HorizonY)
		}
	}
	if st.Coast.LighthouseX <= 0 || st.Coast.LighthouseX >= 40 {
		t.Fatalf("lighthouse x %d out of range", st.Coast.LighthouseX)
	}
	if st.Coast.LighthouseH <= 0 || st.Coast.LighthouseH >= st.City.HorizonY {
		t.Fatalf("lighthouse height %d out of range (horizon %d)", st.Coast.LighthouseH, st.City.HorizonY)
	}
}
