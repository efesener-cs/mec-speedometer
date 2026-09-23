package main

import "testing"

func TestUnitFactor(t *testing.T) {
	tests := []struct {
		unit   string
		factor float64
		suffix string
		ok     bool
	}{
		{"m/s", 1, "m/s", true},
		{"KM/H", 3.6, "km/h", true},
		{"mph", 2.2369362921, "mph", true},
		{"knots", 0, "", false},
	}
	for _, tt := range tests {
		factor, suffix, ok := unitFactor(tt.unit)
		if factor != tt.factor || suffix != tt.suffix || ok != tt.ok {
			t.Errorf("unitFactor(%q) = (%v, %q, %v), want (%v, %q, %v)", tt.unit, factor, suffix, ok, tt.factor, tt.suffix, tt.ok)
		}
	}
}

func TestNameMatchesPrefix(t *testing.T) {
	for _, tc := range []struct {
		actual, wanted string
		want           bool
	}{
		{"MirrorsEdgeCatalyst.exe", "MirrorsEdgeCata.exe", true},
		{"MirrorsEdgeCata.exe", "MirrorsEdgeCatalyst.exe", true},
		{"OtherGame.exe", "MirrorsEdgeCata.exe", false},
	} {
		if got := nameMatches(tc.actual, tc.wanted); got != tc.want {
			t.Errorf("nameMatches(%q, %q) = %v, want %v", tc.actual, tc.wanted, got, tc.want)
		}
	}
}

func TestModuleBaseHandlesPathsWithSpaces(t *testing.T) {
	maps := "7f120000-7f120020 r--p 00000000 08:01 123 /home/player/Steam Library/MirrorsEdgeCatalyst.exe\n" +
		"7f120020-7f121000 r-xp 00002000 08:01 123 /home/player/Steam Library/MirrorsEdgeCatalyst.exe\n"
	got, err := moduleBase(maps, "MirrorsEdgeCata.exe")
	if err != nil {
		t.Fatalf("moduleBase returned error: %v", err)
	}
	if want := uint64(0x7f120000); got != want {
		t.Errorf("moduleBase = %#x, want %#x", got, want)
	}
}
