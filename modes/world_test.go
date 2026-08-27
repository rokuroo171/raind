package modes

import "testing"

func TestNewSkylineDeterministic(t *testing.T) {
	a := NewSkyline(123, 80, 24)
	b := NewSkyline(123, 80, 24)
	if len(a.Buildings) != len(b.Buildings) {
		t.Fatalf("building count differs: %d vs %d", len(a.Buildings), len(b.Buildings))
	}
	for i := range a.Buildings {
		if a.Buildings[i].X != b.Buildings[i].X ||
			a.Buildings[i].W != b.Buildings[i].W ||
			a.Buildings[i].H != b.Buildings[i].H {
			t.Fatalf("building %d differs: %+v vs %+v", i, a.Buildings[i], b.Buildings[i])
		}
	}
	if a.Tallest != b.Tallest {
		t.Fatalf("tallest index differs: %d vs %d", a.Tallest, b.Tallest)
	}
}

func TestNewSkylineFitsBounds(t *testing.T) {
	sk := NewSkyline(7, 30, 16)
	horizon := sk.HorizonY
	if len(sk.Buildings) == 0 {
		t.Fatal("expected buildings")
	}
	for i, b := range sk.Buildings {
		if b.X < 0 || b.X+b.W > 30 {
			t.Fatalf("building %d out of horizontal bounds: %+v", i, b)
		}
		if b.H < 1 || b.H >= horizon {
			t.Fatalf("building %d out of vertical bounds: %+v (horizon %d)", i, b, horizon)
		}
		if len(b.Lit) != b.W*b.H {
			t.Fatalf("building %d lit buffer mismatch: %d != %d", i, len(b.Lit), b.W*b.H)
		}
	}
	if sk.Tallest < 0 || sk.Tallest >= len(sk.Buildings) {
		t.Fatalf("bad tallest index: %d", sk.Tallest)
	}
}

func TestNewSkylineSmallTerminal(t *testing.T) {
	sk := NewSkyline(1, 1, 3)
	if len(sk.Buildings) != 0 {
		t.Fatalf("expected no buildings on a tiny terminal, got %d", len(sk.Buildings))
	}
}
