package ai

import (
	"testing"

	"github.com/gopherd/landlord/poker"
	"github.com/gopherd/log"
)

func assert(t *testing.T, value bool) {
	if !value {
		t.Fail()
		log.Print(1, log.LevelFatal, "assert failed")
	}
}

func TestNewPokerSet(t *testing.T) {
	var pset PokerSet
	pset = NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P3))
	assert(t, pset.raw() == 0x1)
	pset = NewPokerSetWithPoker(poker.NewPoker(0, poker.PJoker1))
	assert(t, pset.raw() == 0x10000000000000)
	pset = NewPokerSetWithPoker(poker.NewPoker(0, poker.PJoker2))
	assert(t, pset.raw() == 0x100000000000000)

	pset = NewBomb(poker.P3)
	assert(t, pset.raw() == 0xF)
	pset = NewBomb(poker.PM2)
	assert(t, pset.raw() == 0xF000000000000)
}

func TestWalk(t *testing.T) {
	var pset PokerSet
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P3)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.PM2)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.PM2)))
	assert(t, pset.Len() == 3)
	cnt := 0
	pset.Walk(func(p poker.Poker) bool {
		cnt++
		return false
	})
	assert(t, cnt == 3)
	cnt = 0
	pset.WalkBlock(func(value poker.Value, block Block) bool {
		if value == poker.P3 {
			assert(t, block.Len() == 1)
			cnt++
		} else if value == poker.PM2 {
			assert(t, block.Len() == 2)
			cnt++
		} else {
			assert(t, block.Len() == 0)
		}
		return false
	})
	assert(t, cnt == 2)
}

func TestRemove(t *testing.T) {
	var pset PokerSet
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P3)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.PM2)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.PM2)))
	pset.Remove(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.PM2)))
	assert(t, pset.Len() == 2)
	cnt := 0
	pset.Walk(func(p poker.Poker) bool {
		cnt++
		return false
	})
	assert(t, cnt == 2)
	cnt = 0
	pset.WalkBlock(func(value poker.Value, block Block) bool {
		if value == poker.P3 {
			assert(t, block.Len() == 1)
			cnt++
		} else if value == poker.PM2 {
			assert(t, block.Len() == 1)
			cnt++
		} else {
			assert(t, block.Len() == 0)
		}
		return false
	})
	assert(t, cnt == 2)
}

func TestContains(t *testing.T) {
	var pset1 PokerSet
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P3)))
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.PM2)))
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.PM2)))

	var pset2 PokerSet
	pset2.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P3)))
	pset2.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.PM2)))

	var pset3 PokerSet
	pset3.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.PM2)))

	assert(t, pset1.Contains(pset2))
	assert(t, pset1.Contains(pset3))
	assert(t, !pset2.Contains(pset3))
}

func TestCount(t *testing.T) {
	var pset PokerSet
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P3)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.PM2)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.PM2)))
	assert(t, pset.Count(poker.P3) == 1)
	assert(t, pset.Count(poker.PM2) == 2)
	assert(t, pset.Count(poker.P4) == 0)
	assert(t, pset.Count(poker.PMA) == 0)
}

func TestAddByValue(t *testing.T) {
	var pset1 PokerSet
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P3)))
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.PM2)))
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.PM2)))

	var pset2 PokerSet
	pset2.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P3)))
	pset2.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.PM2)))

	var pset3 PokerSet
	pset3.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.PM2)))

	assert(t, pset2.AddByValue(pset3, poker.PM2, 1) == pset3)
	assert(t, pset2.AddByValue(pset3, poker.PM2, 2) == emptyPokerSet)
}

func TestBomb(t *testing.T) {
	for bomb, _ := range bombs {
		assert(t, bomb.IsBomb())
	}
}

func TestNormalize(t *testing.T) {
	var pset1 PokerSet
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.P3)))
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.PM2)))
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Diamond, poker.PM2)))

	var pset2 PokerSet
	pset2.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P3)))
	pset2.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.PM2)))
	pset2.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.PM2)))

	assert(t, pset1.Normalize() == pset2)
}

func TestMinValue(t *testing.T) {
	var pset1 PokerSet
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.P4)))
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.PM2)))
	pset1.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Diamond, poker.PM2)))

	assert(t, pset1.MinValue() == poker.P4)
}

func TestMatch(t *testing.T) {
	// [♦4,♥5,♦5,♠6,♥7,♣7,♥8,♦8,♠9,♥9,♣Q,♦Q,♦2]
	var pset PokerSet
	pokers := []poker.Poker{
		poker.NewPoker(poker.Spade, poker.P4),
		poker.NewPoker(poker.Spade, poker.P5),
		poker.NewPoker(poker.Heart, poker.P5),
		poker.NewPoker(poker.Spade, poker.P6),
		poker.NewPoker(poker.Spade, poker.P7),
		poker.NewPoker(poker.Heart, poker.P7),
		poker.NewPoker(poker.Spade, poker.P8),
		poker.NewPoker(poker.Heart, poker.P8),
		poker.NewPoker(poker.Spade, poker.P9),
		poker.NewPoker(poker.Heart, poker.P9),
		poker.NewPoker(poker.Spade, poker.PQ),
		poker.NewPoker(poker.Heart, poker.PQ),
		poker.NewPoker(poker.Spade, poker.PM2),
	}
	for _, p := range pokers {
		pset.Add(NewPokerSetWithPoker(p))
	}
	kinds := pset.Match(Kind{}, Kind{}, DefaultOptions, 256)
	t.Logf("pset %v all kinds: %v", pset, kinds)

	kind := NewKind(3, 2, 0, 0)
	kind.minValue = poker.P3
	kinds = pset.Match(Kind{}, kind, DefaultOptions, 256)
	t.Logf("pset %v all kinds for %v: %v", pset, kind, kinds)
	kinds = pset.Match(kind, Kind{}, DefaultOptions, 256)
	t.Logf("pset %v all kinds for %v: %v", pset, kind, kinds)

	kind.minValue = poker.P7
	kinds = pset.Match(kind, Kind{}, DefaultOptions, 256)
	t.Logf("pset %v all kinds for %v: %v", pset, kind, kinds)

	// [♦3,♥3,♦3,♠4,♥4,♣4,♥6,♦6,♠6,♥7,♣7,♦8,♦9]
	pset.Clear()
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P3)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.P3)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Diamond, poker.P3)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P4)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.P4)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Club, poker.P4)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Spade, poker.P6)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.P6)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Diamond, poker.P6)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Heart, poker.P7)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Club, poker.P7)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Diamond, poker.P8)))
	pset.Add(NewPokerSetWithPoker(poker.NewPoker(poker.Diamond, poker.P9)))
	kinds = pset.Match(Kind{}, Kind{}, DefaultOptions, 256)
	t.Logf("pset %v all %d kinds: %v", pset, len(kinds), kinds)
}
