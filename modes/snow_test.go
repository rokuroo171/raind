package modes

import "testing"

func TestSnowRoofDepositCityOnly(t *testing.T) {
	st := &State{Width: 40, Height: 24, World: WorldCity, Intensity: 0.5}
	st.City = NewSkyline(3, 40, 24)
	st.updateRooftops()
	st.RoofAccum = make([]int, 40)

	roofCol := -1
	for x, r := range st.RoofTop {
		if r >= 0 {
			roofCol = x
			break
		}
	}
	if roofCol < 0 {
		t.Fatal("expected a roofed column")
	}
	st.snowRoofDeposit(roofCol, 40)
	if st.RoofAccum[roofCol] == 0 {
		t.Error("roof deposit should accumulate on a roofed column")
	}

	// a column with no roof must never accumulate snow itself
	for x, r := range st.RoofTop {
		if r < 0 && x != roofCol {
			st.snowRoofDeposit(x, 40)
			if st.RoofAccum[x] != 0 {
				t.Errorf("non-roofed column %d accumulated snow", x)
			}
			break
		}
	}
}

func TestSnowRoofDepositCaps(t *testing.T) {
	st := &State{Width: 40, Height: 24, World: WorldCity, Intensity: 1.0}
	st.City = NewSkyline(3, 40, 24)
	st.updateRooftops()
	st.RoofAccum = make([]int, 40)

	roofCol := -1
	for x, r := range st.RoofTop {
		if r >= 0 {
			roofCol = x
			break
		}
	}
	limit := 2 + int(st.Intensity*3)
	for i := 0; i < limit+10; i++ {
		st.snowRoofDeposit(roofCol, 40)
	}
	// the cap applies per column, and the deposit spreads across the roof
	for x, v := range st.RoofAccum {
		if v > limit {
			t.Errorf("roof column %d accumulated %d, cap is %d", x, v, limit)
		}
	}
}

func TestSnowCoastRockCaps(t *testing.T) {
	st := &State{Width: 40, Height: 24, World: WorldCoast, Intensity: 1.0}
	st.City = NewSkyline(3, 40, 24)
	st.initCoast(40, 24)
	st.AccumRow = make([]int, 40)

	lx := st.Coast.LighthouseX
	limit := 1 + int(st.Intensity*3)
	for i := 0; i < limit+10; i++ {
		st.snowGroundDeposit(lx, 10, 40)
	}
	if st.AccumRow[lx] > limit {
		t.Errorf("rock snow should cap at %d, got %d", limit, st.AccumRow[lx])
	}

	// flakes far from the rock fade into the sea: no accumulation
	st.snowGroundDeposit(0, 10, 40)
	if st.AccumRow[0] != 0 {
		t.Errorf("open sea should not accumulate, got %d", st.AccumRow[0])
	}
}

func TestSnowMaxAccumScalesWithIntensity(t *testing.T) {
	st := &State{Intensity: 0.1}
	low := st.snowMaxAccum(24)
	st.Intensity = 1.0
	high := st.snowMaxAccum(24)
	if high < low {
		t.Errorf("whiteout should bank higher than a dusting: %d < %d", high, low)
	}
	if low < 1 || high < 1 {
		t.Errorf("banks should be at least 1 cell: low %d high %d", low, high)
	}
}
