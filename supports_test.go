package chalk

import "testing"

func TestSupportsPredicates(t *testing.T) {
	defer SetLevel(LevelNone)

	cases := []struct {
		level                            Level
		color, basic, c256, trueColorHas bool
	}{
		{LevelNone, false, false, false, false},
		{LevelBasic, true, true, false, false},
		{Level256, true, true, true, false},
		{LevelTrueColor, true, true, true, true},
	}
	for _, c := range cases {
		SetLevel(c.level)
		if got := SupportsColor(); got != c.color {
			t.Errorf("level %d: SupportsColor = %v; want %v", c.level, got, c.color)
		}
		if got := HasBasic(); got != c.basic {
			t.Errorf("level %d: HasBasic = %v; want %v", c.level, got, c.basic)
		}
		if got := Has256(); got != c.c256 {
			t.Errorf("level %d: Has256 = %v; want %v", c.level, got, c.c256)
		}
		if got := HasTrueColor(); got != c.trueColorHas {
			t.Errorf("level %d: HasTrueColor = %v; want %v", c.level, got, c.trueColorHas)
		}
	}
}
