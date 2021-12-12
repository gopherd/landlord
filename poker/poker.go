package poker

import (
	"bytes"
	"sort"
	"strconv"
)

type Suit uint8

const (
	Spade   Suit = 0
	Heart   Suit = 1
	Club    Suit = 2
	Diamond Suit = 3
)

type Value uint8

const (
	InvalidPokerValue Value = 0

	PA      Value = 1
	P2      Value = 2
	P3      Value = 3
	P4      Value = 4
	P5      Value = 5
	P6      Value = 6
	P7      Value = 7
	P8      Value = 8
	P9      Value = 9
	P10     Value = 10
	PJ      Value = 11
	PQ      Value = 12
	PK      Value = 13
	PMA     Value = 14
	PM2     Value = 15
	PJoker1 Value = 16
	PJoker2 Value = 17
)

func (v Value) String() string {
	if v == P10 {
		return "X"
	}
	if v >= P2 && v < P10 {
		return strconv.Itoa(int(v))
	} else if v == PM2 {
		return "2"
	} else if v == PA || v == PMA {
		return "A"
	} else if v == PJ {
		return "J"
	} else if v == PQ {
		return "Q"
	} else if v == PK {
		return "K"
	} else if v == PJoker1 {
		return "#"
	} else if v == PJoker2 {
		return "$"
	}
	return "-"
}

func FormatValues(values []Value) string {
	var head bytes.Buffer
	var body bytes.Buffer
	var foot bytes.Buffer
	head.WriteString("┏")
	body.WriteString("┃")
	foot.WriteString("┗")
	for i := range values {
		if i > 0 {
			head.WriteString("┳")
			body.WriteString("┃")
			foot.WriteString("┻")
		}
		head.WriteString("━")
		body.WriteString(values[i].String())
		foot.WriteString("━")
	}
	head.WriteString("┓")
	body.WriteString("┃")
	foot.WriteString("┛")
	return head.String() + "\n" + body.String() + "\n" + foot.String()
}

const (
	Joker1 Poker = Poker(Spade<<5) | Poker(PJoker1)
	Joker2 Poker = Poker(Spade<<5) | Poker(PJoker2)
)

type Poker uint8

func NewPoker(suit Suit, value Value) Poker {
	return Poker(int(suit<<5) | int(value))
}

func (p Poker) Int32() int32 { return int32(p) }

func (p Poker) Suit() Suit {
	return Suit((p >> 5) & 0x3)
}

func (p Poker) Value() Value {
	return Value(p & 0x1F)
}

func (p Poker) Less(p2 Poker) bool {
	return p.Value() < p2.Value() || (p.Value() == p2.Value() && p.Suit() > p2.Suit())
}

func (p Poker) Greater(p2 Poker) bool {
	return p.Value() > p2.Value() || (p.Value() == p2.Value() && p.Suit() < p2.Suit())
}

func (p Poker) IsJoker1() bool {
	return p.Value() == PJoker1
}

func (p Poker) IsJoker2() bool {
	return p.Value() == PJoker2
}

func (p Poker) IsJoker() bool {
	value := p.Value()
	return value == PJoker1 || value == PJoker2
}

func (p Poker) String() string {
	suit := p.Suit()
	value := p.Value()
	ret := ""
	if !p.IsJoker() {
		switch suit {
		case Spade:
			ret += "♠"
		case Heart:
			ret += "♥"
		case Club:
			ret += "♣"
		case Diamond:
			ret += "♦"
		}
	}
	ret += value.String()
	return ret
}

type PokerSlice []Poker

func (ps *PokerSlice) Add(p Poker) {
	*ps = append(*ps, p)
}

func (ps *PokerSlice) Remove(p Poker) bool {
	for i := 0; i < len(*ps); i++ {
		if (*ps)[i] == p {
			*ps = append((*ps)[:i], (*ps)[i+1:]...)
			return true
		}
	}
	return false
}

func (ps *PokerSlice) Clear() {
	*ps = (*ps)[0:0]
}

func (ps PokerSlice) Len() int           { return len(ps) }
func (ps PokerSlice) Less(i, j int) bool { return ps[i].Less(ps[j]) }
func (ps PokerSlice) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }

func (ps PokerSlice) Format(prefix, sep, suffix string) string {
	var buf bytes.Buffer
	buf.WriteString(prefix)
	for i, p := range ps {
		if i > 0 {
			buf.WriteString(sep)
		}
		buf.WriteString(p.String())
	}
	buf.WriteString(suffix)
	return buf.String()
}

func (ps PokerSlice) String() string {
	return ps.Format("[", ",", "]")
}

func IntsToPokerSlice(pokers []int) PokerSlice {
	list := make([]Poker, 0, len(pokers))
	for _, p := range pokers {
		list = append(list, Poker(p))
	}
	return PokerSlice(list)
}

func ToPokers(pokers []int32) []Poker {
	list := make([]Poker, 0, len(pokers))
	for _, p := range pokers {
		list = append(list, Poker(p))
	}
	return list
}

func ToInt32s(pokers []Poker) []int32 {
	int32s := make([]int32, 0)
	for _, poker := range pokers {
		int32s = append(int32s, int32(poker))
	}
	return int32s
}

func Sort(pokers []Poker) {
	sort.Sort(PokerSlice(pokers))
}

func SortValues(values []Value) {
	sort.SliceStable(values, func(i, j int) bool {
		return values[i] < values[j]
	})
}

// 牌型
type Type int

const (
	None Type = 0 // 不出

	// 单牌和顺子
	Single1  Type = 101
	Single5  Type = 105
	Single6  Type = 106
	Single7  Type = 107
	Single8  Type = 108
	Single9  Type = 109
	Single10 Type = 110
	Single11 Type = 111
	Single12 Type = 112

	// 对子和连对
	Double1  Type = 201
	Double3  Type = 203
	Double4  Type = 204
	Double5  Type = 205
	Double6  Type = 206
	Double7  Type = 207
	Double8  Type = 208
	Double9  Type = 209
	Double10 Type = 210

	// 3带1
	ThreeSingle1 Type = 301
	ThreeSingle2 Type = 302
	ThreeSingle3 Type = 303
	ThreeSingle4 Type = 304
	ThreeSingle5 Type = 305

	// 3带2
	ThreeDouble1 Type = 401
	ThreeDouble2 Type = 402
	ThreeDouble3 Type = 403
	ThreeDouble4 Type = 404

	// 3不带
	Three1 Type = 501
	Three2 Type = 502
	Three3 Type = 503
	Three4 Type = 504
	Three5 Type = 505
	Three6 Type = 506

	// 4带2单
	FourSingle1 Type = 601

	// 4带2对
	FourDouble1 Type = 701

	// 炸弹和火箭
	Bomb   Type = 1801
	Rocket Type = 2901
)
