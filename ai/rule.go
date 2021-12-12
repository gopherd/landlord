package ai

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/gopherd/doge/bits"
	"github.com/gopherd/doge/math/mathutil"

	"github.com/gopherd/landlord/poker"
)

const (
	minPokerValue = poker.P3
	maxPokerValue = poker.PJoker2
	numPokerValue = int(maxPokerValue - minPokerValue + 1)
	numValidBits  = numPokerValue * 4
)

// 游戏细则
type Options struct {
	// 是否允许三带对子
	CanTrioWithPair bool `json:"can_trio_with_pair"`
	// 是否允许4个2带牌
	CanFourTwoWithKickers bool `json:"can_four_two_with_kickers"`
	// 是否允许带牌和主体部分的牌相同,比如333444 是否允许带3或4
	CanKickerInBody bool `json:"can_kicker_in_body"`
	// 是否允许3张不带
	CanTrioWithoutKicker bool `json:"can_trio_without_kicker"`
	// 是否允许航天飞机(类似于飞机,不过是4张,如: 33334444+5678)
	CanSpaceShuttle bool `json:"can_space_shuttle"`
	// 是否允许带牌重复: 比如 333444+55, 333444555+778, 33334444+5567
	CanRepeatKicker bool `json:"can_repeat_kicker"`
	// 是否允许带牌时带王
	CanJokerAsKicker bool `json:"can_joker_as_kicker"`
	// 普通炸弹倍数
	MultipleOfBomb int `json:"multiple_of_bomb"`
	// 火箭(王炸)倍数
	MultipleOfRocket int `json:"multiple_of_rocket"`
	// 顺子最少长度
	MinLengthOfChain int `json:"min_length_of_chain"`
	// 连对最少长度
	MinLengthOfPairChain int `json:"min_length_of_pair_chain"`
}

var DefaultOptions = Options{
	CanTrioWithPair:       true,
	CanFourTwoWithKickers: true,
	CanKickerInBody:       true,
	CanTrioWithoutKicker:  true,
	CanSpaceShuttle:       false,
	CanRepeatKicker:       true,
	CanJokerAsKicker:      false,
	MultipleOfBomb:        2,
	MultipleOfRocket:      2,
	MinLengthOfChain:      5,
	MinLengthOfPairChain:  3,
}

// 按位表示的扑克牌集合
// 每 4 bits 为一个块,分别表示一个扑克面值的 4 种花色的牌
// 从最低位开始的 15 个块(60 bits) 分表表示牌值 3,4,...,A,2,Joker1,Joker2
// 最高 4 bits 预留,可以表示任何扩展数据,代码里用 score 表示它的值(0~15)
//
// 1 1 1 1 1 1 1 1
// ------- ------- (第1个8位)
//    4       3
//
// ...............
//
// 1 1 1 1 1 1 1 1
// ------- ------- (第6个8位)
//    A       K
//
// 1 1 1 1 1 1 1 1
// ------- ------- (第7个8位)
// Joker1     2
//
// 1 1 1 1 1 1 1 1
// ------- ------- (第8个8位)
//  score  Joker2
type PokerSet uint64

const emptyPokerSet PokerSet = 0

var joker1 = NewPokerSetWithPoker(poker.Joker1)
var joker2 = NewPokerSetWithPoker(poker.Joker2)
var rocket = joker1.Add(joker2)
var bombs = map[PokerSet]poker.Value{
	NewBomb(poker.P3):  poker.P3,
	NewBomb(poker.P4):  poker.P4,
	NewBomb(poker.P5):  poker.P5,
	NewBomb(poker.P6):  poker.P6,
	NewBomb(poker.P7):  poker.P7,
	NewBomb(poker.P8):  poker.P8,
	NewBomb(poker.P9):  poker.P9,
	NewBomb(poker.P10): poker.P10,
	NewBomb(poker.PJ):  poker.PJ,
	NewBomb(poker.PQ):  poker.PQ,
	NewBomb(poker.PK):  poker.PK,
	NewBomb(poker.PMA): poker.PMA,
	NewBomb(poker.PM2): poker.PM2,
}

func (pset PokerSet) raw() uint64 { return uint64(pset) }

func NewPokerSetWithPoker(p poker.Poker) PokerSet {
	return PokerSet(1 << (uint(p.Value()-minPokerValue)<<2 + uint(p.Suit())))
}

func NewPokerSetWithInt32s(pokers []int32) PokerSet {
	var ret PokerSet
	for _, p := range pokers {
		ret.Add(NewPokerSetWithPoker(poker.Poker(p)))
	}
	return ret
}

func NewBomb(value poker.Value) PokerSet {
	return PokerSet(uint64(0xF) << (uint(value-minPokerValue) << 2))
}

// 扑克牌集合字符串输出
func (pset PokerSet) String() string {
	var buf bytes.Buffer
	buf.WriteByte('[')
	count := 0
	for i := uint(0); i < 60; i++ {
		if (pset>>i)&0x1 != 0 {
			suit := poker.Suit(i % 4)
			value := poker.Value(i/4) + minPokerValue
			if count > 0 {
				buf.WriteByte(',')
			}
			count++
			buf.WriteString(poker.NewPoker(suit, value).String())
		}
	}
	buf.WriteByte(']')
	return buf.String()
}

// 扑克牌集合字符串输出
func (pset PokerSet) StringWithoutSuit() string {
	var buf bytes.Buffer
	buf.WriteByte('[')
	count := 0
	for i := uint(0); i < 60; i++ {
		if (pset>>i)&0x1 != 0 {
			value := poker.Value(i/4) + minPokerValue
			if count > 0 {
				buf.WriteByte(',')
			}
			count++
			buf.WriteString(value.String())
		}
	}
	buf.WriteByte(']')
	return buf.String()
}

// 获取扑克牌张数
func (pset PokerSet) Len() int { return bits.Count64(pset.raw()) }

func (pset PokerSet) GetValues() []poker.Value {
	var values = make([]poker.Value, 0, pset.Len())
	for i := uint(0); i < 60; i++ {
		if (pset>>i)&0x1 != 0 {
			values = append(values, poker.Value(i/4)+minPokerValue)
		}
	}
	return values
}

// 判断牌集是否为空
func (pset PokerSet) Empty() bool { return pset == 0 }

// 扑克牌遍历

type PokerVisitor func(poker.Poker) (terminate bool)

func (pset PokerSet) Walk(visitor PokerVisitor) bool {
	for i := uint(0); i < uint(numValidBits); i++ {
		if (pset>>i)&0x1 == 0 {
			continue
		}
		suit := poker.Suit(i & 0x3)
		value := poker.Value(i>>2) + minPokerValue
		if terminate := visitor(poker.NewPoker(suit, value)); terminate {
			return true
		}
	}
	return false
}

// 输出成整数数组
func (pset PokerSet) ToInt32s(target []int32) []int32 {
	if len(target) == 0 {
		target = make([]int32, 0, pset.Len())
	}
	pset.Walk(func(p poker.Poker) bool {
		target = append(target, p.Int32())
		return false
	})
	return target
}

// 扑克牌值遍历

type Block uint8

func (b Block) Len() int { return bits.Count8(uint8(b)) }

type BlockVisitor func(poker.Value, Block) (terminate bool)

func (pset PokerSet) WalkBlock(visitor BlockVisitor) bool {
	for i := uint(0); i+4 <= uint(numValidBits); i += 4 {
		value := poker.Value(i>>2) + minPokerValue
		block := Block(uint8(pset>>i) & 0xf)
		if terminate := visitor(value, block); terminate {
			return true
		}
	}
	return false
}

// 按各种牌剩余张数表示的整数数组
func (pset PokerSet) Nums() []int {
	ret := make([]int, 0, numPokerValue)
	pset.WalkBlock(func(value poker.Value, block Block) bool {
		ret = append(ret, block.Len())
		return false
	})
	return ret
}

// 增加扑克牌
func (pset *PokerSet) Add(pset2 PokerSet) PokerSet {
	*pset = *pset | pset2
	return *pset
}

// 删除扑克牌,被删除的集合必须是当前集合的子集,否则返回false
func (pset *PokerSet) Remove(pset2 PokerSet) bool {
	s1 := *pset
	s2 := pset2
	if s1&s2 != s2 {
		return false
	}
	*pset = *pset - s2
	return true
}

// 清空牌集合
func (pset *PokerSet) Clear() { *pset = emptyPokerSet }

// 判断是否包含另一个扑克牌集合
func (pset PokerSet) Contains(pset2 PokerSet) bool { return pset&pset2 == pset2 }

// 判断包含某张扑克牌
func (pset PokerSet) HasPoker(p poker.Poker) bool {
	return pset&NewPokerSetWithPoker(p) != 0
}

// 计算某个面值(如 3,J,K...)的扑克牌数量
func (pset PokerSet) Count(v poker.Value) int {
	return bits.Count8(uint8((uint64(pset) >> (uint64(v-minPokerValue) << 2)) & 0xF))
}

// 从指定的集合 from 中添加 n 个值为 value 的扑克
func (pset *PokerSet) AddByValue(from PokerSet, value poker.Value, n int) PokerSet {
	var added PokerSet
	index := uint64(value-minPokerValue) << 2
	for i := uint64(0); i < 4; i++ {
		if n <= 0 {
			break
		}
		p := PokerSet(1 << (index + i))
		if (p&(from) != 0) && (p&(*pset) == 0) {
			added |= p
			n--
		}
	}
	if n > 0 {
		return emptyPokerSet
	}
	*pset = *pset | added
	return added
}

func (pset PokerSet) Find(target PokerSet) PokerSet {
	var result PokerSet
	if target.Empty() {
		return result
	}
	target.WalkBlock(func(value poker.Value, block Block) bool {
		n := block.Len()
		if n > 0 {
			pset.Remove(result.AddByValue(pset, value, n))
		}
		return false
	})
	return result
}

// 判定是否是双王
func (pset PokerSet) IsRocket() bool {
	return pset == rocket
}

// 判断是否普通炸弹
func (pset PokerSet) IsBomb() bool {
	return bombs[pset] > 0
}

// 牌集正则化,去除花色差异
func (pset PokerSet) Normalize() PokerSet {
	var ret PokerSet
	for i := 0; i < 8; i++ {
		offset := uint(i) << 3
		ret |= PokerSet(bits.Normalize4(uint8((pset>>offset)&0xFF))) << offset
	}
	return ret
}

// 获取最小扑克牌值
func (pset PokerSet) MinValue() poker.Value {
	for i := uint(0); i <= uint(numValidBits); i += 4 {
		if ((pset >> i) & 0xF) != 0 {
			return poker.Value(i>>2) + minPokerValue
		}
	}
	return poker.Value(0)
}

func (pset PokerSet) matchBody(w, h int8, begin poker.Value, opts Options) []PokerSet {
	var (
		ret   []PokerSet
		start = poker.Value(0)
		cur   = emptyPokerSet
	)
	begin = begin + 1
	if w == 2 && h == 1 {
		// 宽度为 2 且高度为 1 的只有火箭
		if begin <= poker.PJoker1 {
			begin = poker.PJoker1
		} else {
			return ret
		}
	}
	for value := begin; value <= maxPokerValue; value++ {
		if value < minPokerValue {
			continue
		}
		if w > 1 && value >= poker.PM2 {
			break
		}
		if h == 4 {
			if value > poker.PM2 || (value == poker.PM2 && !opts.CanFourTwoWithKickers) {
				break
			}
		}
		if !cur.AddByValue(pset, value, int(h)).Empty() {
			if start == 0 {
				start = value
			}
			if int(value-start+1) == int(w) {
				ret = append(ret, cur)
				offset := uint(start-minPokerValue+1) << 2
				cur = (cur >> offset) << offset
				start++
			}
		} else {
			// 中断了,重新开始
			cur = emptyPokerSet
			start = 0
		}
	}
	return ret
}

func (pset PokerSet) match(kind Kind, strict bool, opt Options, ret []Kind, limit int) []Kind {
	var bodies = pset.matchBody(kind.width, kind.height, kind.minValue, opt)
	for _, body := range bodies {
		if kind.hasKicker() {
			// 选取带牌
			var (
				kickers         [][]PokerSet
				nums            []int
				prevKickerValue poker.Value
				remain          = pset
			)
			remain.Remove(body)
			for value := minPokerValue; value <= maxPokerValue; {
				if !opt.CanJokerAsKicker && value >= poker.PJoker1 {
					break
				}
				if !opt.CanKickerInBody && body.Count(value) > 0 {
					value++
					continue
				}
				var added PokerSet
				added = added.AddByValue(remain, value, int(kind.kickerHeight))
				if !added.Empty() {
					if value == prevKickerValue {
						kickers[len(kickers)-1] = append(kickers[len(kickers)-1], added)
						nums[len(nums)-1]++
					} else {
						prevKickerValue = value
						kickers = append(kickers, []PokerSet{added})
						nums = append(nums, 1)
					}
					remain.Remove(added)
				}
				if added.Empty() ||
					!opt.CanRepeatKicker ||
					remain.Count(value) < int(kind.kickerHeight) {
					value++
				}
			}
			multiCombSet := mathutil.MultiCombSet(nums, int(kind.kickerWidth))
			if len(multiCombSet) == 0 {
				continue
			}
			for _, ins := range multiCombSet {
				var kicker PokerSet
				for i, n := range ins {
					for j := 0; j < n; j++ {
						kicker.Add(kickers[i][j])
					}
				}
				ret = append(ret, kind.extend(body, kicker))
			}
		} else {
			ret = append(ret, kind.extend(body, emptyPokerSet))
		}
		if len(ret) >= limit {
			return ret
		}
	}

	// 非严格模式尝试匹配炸弹和火箭
	if !strict {
		if !kind.IsBomb() && !kind.IsRocket() {
			// 如果当前不是炸弹则尝试匹配炸弹
			// 如果已经是炸弹,那么在 body 匹配中就已经完成,这里就不需要了
			for value := minPokerValue; value <= poker.PM2; value++ {
				var body PokerSet
				if body.AddByValue(pset, value, 4) == 4 {
					kind2 := kindsMap[poker.Bomb].extend(body, emptyPokerSet)
					ret = append(ret, kind2)
					if len(ret) >= limit {
						return ret
					}
				}
			}
		}
		if !kind.IsRocket() {
			// 如果当前不是火箭则尝试匹配火箭
			// 如果已经是火箭,那么在 body 匹配中就已经完成,这里就不需要了
			if pset.Contains(rocket) {
				kind2 := kindsMap[poker.Rocket].extend(rocket, emptyPokerSet)
				ret = append(ret, kind2)
			}
		}
	}

	return ret
}

func (pset PokerSet) Match(kind1, kind2 Kind, opt Options, limit int) []Kind {
	kind := kind2
	if kind.Len() == 0 {
		kind = kind1
	}

	// 前两手都没有人出牌,则玩家可以选择任意合法牌型出牌
	if kind.Len() == 0 {
		ret := make([]Kind, 0, pset.Len()*2)
		for _, k := range kindsMap {
			if k.Len() == 0 {
				continue
			}
			ret = pset.match(k, true, opt, ret, limit)
			if len(ret) >= limit {
				break
			}
		}
		return ret
	}

	// 否则,玩家只能选择能管上 ``kind'' 的牌型出牌或者弃牌
	ret := make([]Kind, 0, 8)
	ret = pset.match(kind, false, opt, ret, limit)
	ret = append(ret, Kind{})
	return ret
}

// 扑克牌型
type Kind struct {
	// 主干部分的宽和高,比如
	// 单张的 width=1, height=1
	// 对子的 width=1, height=2
	// 334455 的 width=3, height=2
	// 3334 的 width=1, height=3
	// 33344456 的 width=2, height=3
	// 小王+大王 的 width=2, height=1
	width, height int8

	// 带牌部分的宽和高,比如
	// 333 的 kickerWidth=0, kickerHeight=0
	// 3334 的 kickerWidth=1, kickerHeight=1
	// 33344 的 kickerWidth=1, kickerHeight=2
	// 555567 的 kickerWidth=2, kickerHeight=1
	// 55556677 的 kickerWidth=2, kickerHeight=2
	kickerWidth, kickerHeight int8

	// 主干部分的最小牌面值
	minValue poker.Value

	// 主干部分的牌
	body PokerSet

	// 带牌部分的牌
	kicker PokerSet

	// 附带参数
	ext int
}

func (kind Kind) String() string {
	if kind.body.Len() > 0 {
		if kind.kicker.Len() > 0 {
			return fmt.Sprintf("{%v + %v}", kind.body, kind.kicker)
		}
		return fmt.Sprintf("{%v}", kind.body)
	}
	if kind.Len() == 0 {
		return "{}"
	}
	if kind.minValue == 0 {
		return fmt.Sprintf("{w: %d, h: %d, kw: %d, kh: %d}",
			kind.width, kind.height, kind.kickerWidth, kind.kickerHeight)
	}
	return fmt.Sprintf("{w: %d, h: %d, kw: %d, kh: %d, min: %d}",
		kind.width, kind.height, kind.kickerWidth, kind.kickerHeight, kind.minValue)
}

func NewKind(width, height, kickerWidth, kickerHeight int8) Kind {
	return Kind{
		width:        width,
		height:       height,
		kickerWidth:  kickerWidth,
		kickerHeight: kickerHeight,
	}
}

func (kind Kind) extend(body, kicker PokerSet) Kind {
	kind2 := kind
	kind2.minValue = body.MinValue()
	kind2.body = body
	kind2.kicker = kicker
	return kind2
}

// 牌型的形状描述
func (kind Kind) shape() uint32 {
	return uint32(kind.width)<<24 |
		uint32(kind.height)<<16 |
		uint32(kind.kickerWidth)<<8 |
		uint32(kind.kickerHeight)
}

func (kind Kind) hasKicker() bool { return kind.kickerWidth > 0 && kind.kickerHeight > 0 }

func (kind Kind) IsRocket() bool {
	return kind.width == 2 && kind.height == 1 && !kind.hasKicker()
}

func (kind Kind) IsBomb() bool {
	return kind.width == 1 && kind.height == 4 && !kind.hasKicker()
}

func (kind Kind) Pokers() PokerSet {
	ret := kind.body
	ret.Add(kind.kicker)
	return ret
}

func (kind Kind) Len() int {
	return int(kind.height*kind.width + kind.kickerHeight*kind.kickerWidth)
}

func (kind Kind) Equal(kind2 Kind) bool {
	return kind.shape() == kind2.shape() &&
		kind.Pokers().Normalize() == kind2.Pokers().Normalize()
}

func (kind Kind) Type() poker.Type {
	return kindsRevMap[kind.shape()]
}

var kindsMap = map[poker.Type]Kind{
	poker.None:         NewKind(0, 0, 0, 0),
	poker.Single1:      NewKind(1, 1, 0, 0),
	poker.Single5:      NewKind(5, 1, 0, 0),
	poker.Single6:      NewKind(6, 1, 0, 0),
	poker.Single7:      NewKind(7, 1, 0, 0),
	poker.Single8:      NewKind(8, 1, 0, 0),
	poker.Single9:      NewKind(9, 1, 0, 0),
	poker.Single10:     NewKind(10, 1, 0, 0),
	poker.Single11:     NewKind(11, 1, 0, 0),
	poker.Single12:     NewKind(12, 1, 0, 0),
	poker.Double1:      NewKind(1, 2, 0, 0),
	poker.Double3:      NewKind(3, 2, 0, 0),
	poker.Double4:      NewKind(4, 2, 0, 0),
	poker.Double5:      NewKind(5, 2, 0, 0),
	poker.Double6:      NewKind(6, 2, 0, 0),
	poker.Double7:      NewKind(7, 2, 0, 0),
	poker.Double8:      NewKind(8, 2, 0, 0),
	poker.Double9:      NewKind(9, 2, 0, 0),
	poker.Double10:     NewKind(10, 2, 0, 0),
	poker.ThreeSingle1: NewKind(1, 3, 1, 1),
	poker.ThreeSingle2: NewKind(2, 3, 2, 1),
	poker.ThreeSingle3: NewKind(3, 3, 3, 1),
	poker.ThreeSingle4: NewKind(4, 3, 4, 1),
	poker.ThreeSingle5: NewKind(5, 3, 5, 1),
	poker.ThreeDouble1: NewKind(1, 3, 1, 2),
	poker.ThreeDouble2: NewKind(2, 3, 2, 2),
	poker.ThreeDouble3: NewKind(3, 3, 3, 2),
	poker.ThreeDouble4: NewKind(4, 3, 4, 2),
	poker.Three1:       NewKind(1, 3, 0, 0),
	poker.Three2:       NewKind(2, 3, 0, 0),
	poker.Three3:       NewKind(3, 3, 0, 0),
	poker.Three4:       NewKind(4, 3, 0, 0),
	poker.Three5:       NewKind(5, 3, 0, 0),
	poker.Three6:       NewKind(6, 3, 0, 0),
	poker.FourSingle1:  NewKind(1, 4, 2, 1),
	poker.FourDouble1:  NewKind(1, 4, 2, 2),
	poker.Bomb:         NewKind(1, 4, 0, 0),
	poker.Rocket:       NewKind(2, 1, 0, 0),
}

var kindsRevMap map[uint32]poker.Type

func init() {
	kindsRevMap = make(map[uint32]poker.Type)
	for k, v := range kindsMap {
		kindsRevMap[v.shape()] = k
	}
}

/*
func (kind *Kind) match(pokers *pb.Landlord2PlayPokers) bool {
	if pokers == nil {
		return kind.Len() == 0
	}
	actualSize := len(pokers.Pokers)
	if actualSize != kind.Len() {
		return false
	}
	if actualSize == 0 {
		return true
	}
	if kind.width == 2 && kind.height == 1 {
		// 这只能是火箭
		p0 := poker.Poker(pokers.Pokers[0])
		p1 := poker.Poker(pokers.Pokers[1])
		v0 := p0.Value()
		v1 := p1.Value()
		kind.body.Add(NewPokerSetWithPoker(p0))
		kind.body.Add(NewPokerSetWithPoker(p1))
		kind.minValue = v0
		return v0 == poker.PJoker1 && v1 == poker.PJoker2
	}
	// 检测主体部分
	prevValue := poker.Value(0)
	for l := 0; l < int(kind.width); l++ {
		value := poker.Poker(pokers.Pokers[l*int(kind.height)]).Value()
		if l > 0 && value != prevValue+1 {
			return false
		}
		prevValue = value
		for h := 0; h < int(kind.height); h++ {
			p := poker.Poker(pokers.Pokers[l*int(kind.height)+h])
			currValue := p.Value()
			kind.body.Add(NewPokerSetWithPoker(p))
			if h == 0 && l == 0 {
				kind.minValue = currValue
			}
			if currValue != prevValue {
				return false
			}
		}
	}
	// 检测带牌部分
	if !kind.hasKicker() {
		// 没有带牌
		return true
	}
	startIndex := int(kind.width * kind.height)
	hasJoker1 := false
	hasJoker2 := false
	for l := 0; l < int(kind.kickerWidth); l++ {
		firstValue := poker.Value(0)
		for h := 0; h < int(kind.kickerHeight); h++ {
			index := startIndex + l*int(kind.kickerHeight) + h
			p := poker.Poker(pokers.Pokers[index])
			kind.kicker.Add(NewPokerSetWithPoker(p))
			currValue := p.Value()
			if h > 0 && currValue != firstValue {
				return false
			}
			if p.IsJoker1() {
				hasJoker1 = true
			} else if p.IsJoker2() {
				hasJoker2 = true
			}
			firstValue = currValue
		}
	}
	if hasJoker1 && hasJoker2 {
		return false
	}

	return true
}

func kindof(pokers *pb.Landlord2PlayPokers) *Kind {
	typ := pb.Landlord2Type_None
	if pokers != nil {
		typ = pb.Landlord2Type(pokers.GetType())
	}
	kind, ok := kindsMap[typ]
	if !ok {
		return nil
	}
	if kind.match(pokers) {
		return &kind
	}
	return nil
}
*/

// 玩家位置

const NumPlayer = 3

type Position int8

const BadPosition Position = -1

func (pos Position) Valid() bool    { return pos >= 0 && pos < NumPlayer }
func (pos Position) Value() int     { return int(pos) }
func (pos Position) Next() Position { return Position((pos + 1) % NumPlayer) }
func (pos Position) Prev() Position { return Position((pos + NumPlayer - 1) % NumPlayer) }

func (pos Position) Role(landlord Position) string {
	if landlord.Prev() == pos {
		return "P"
	} else if landlord.Next() == pos {
		return "N"
	}
	return "L"
}

func (pos Position) IsFriend(landlord, player Position) bool {
	return player == pos || (player != landlord && pos != landlord)
}

// 游戏状态
type State struct {
	// 各玩家剩余牌的正则化表示
	pokers [NumPlayer]PokerSet
	// 地主位置
	landlord Position
	// 出牌游戏过程累计倍数
	multi int16
	// 地主出牌次数
	landlordPlayTimes int8
	// 农民出牌次数
	farmerPlayTimes int8
}

func NewState(pokers [NumPlayer]PokerSet, landlord Position) State {
	s := State{
		landlord: landlord,
		multi:    1,
	}
	for i := range s.pokers {
		s.pokers[i] = pokers[i].Normalize()
	}
	return s
}

func (state State) String() string {
	var buf bytes.Buffer
	for i, pokers := range state.pokers {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprintf(&buf, "p%d: %v", i, pokers)
	}
	return buf.String()
}

func (state *State) Copy(from State) {
	copy(state.pokers[:], from.pokers[:])
	state.landlord = from.landlord
	state.multi = from.multi
	state.landlordPlayTimes = from.landlordPlayTimes
	state.farmerPlayTimes = from.farmerPlayTimes
}

func (state State) NumPokers() int {
	num := 0
	for _, pokers := range state.pokers {
		num += pokers.Len()
	}
	return num
}

func (state State) Winner() Position {
	for player, pokers := range state.pokers {
		if pokers.Empty() {
			return Position(player)
		}
	}
	return BadPosition
}

func (state State) Gameover() bool { return state.Winner().Valid() }

func (state State) IsSpring(winner Position) bool {
	if winner == state.landlord {
		return state.farmerPlayTimes == 0
	}
	return state.landlordPlayTimes <= 1
}

// 出牌动作
type Action struct {
	// 玩家位置
	player Position
	// 选择出的牌(或不出)
	kind Kind
	// 先验概率
	prob float64
}

func (act Action) String() string {
	return fmt.Sprintf("{pos: %v, kind: %v, prob: %.4g}", act.player, act.kind, act.prob)
}

// 从状态 ``from'' 执行动作,返回执行后新的状态
func (act Action) Do(from State) State {
	var to State
	to.Copy(from)

	// 删掉出的牌
	removed := act.kind.Pokers().Normalize()
	to.pokers[act.player].Remove(removed)
	to.pokers[act.player] = to.pokers[act.player].Normalize()

	// 累计出牌次数
	if act.player == from.landlord {
		to.landlordPlayTimes++
	} else {
		to.farmerPlayTimes++
	}

	// 累计炸弹倍数
	if act.kind.IsBomb() || act.kind.IsRocket() {
		to.multi <<= 1
	}

	return to
}

func (act Action) Equal(act2 Action) bool {
	return act.player == act2.player && act.kind.Equal(act2.kind)
}

type PolicyFunc func(root *Node) ([]Action, float64, int)
type RolloutFunc func(root, leaf *Node) float64

// 获取所有合法操作
func getLegalActions(node *Node) ([]Action, float64, int) {
	var (
		kind1 Kind
		kind2 = node.action.kind
		next  = node.action.player.Next()
	)
	if node.parent != nil {
		kind1 = node.parent.action.kind
	}

	kinds := node.state.pokers[next].Match(kind1, kind2, DefaultOptions, 256)

	// 计算权重,如果所有权重都为 0,则所有权重都加 1
	total := float64(0)
	for _, kind := range kinds {
		total += float64(kind.ext)
	}
	added := float64(0)
	if total == 0 {
		added = 1
		total += added * float64(len(kinds))
	}

	// 创建 Actions
	var actions []Action
	for _, kind := range kinds {
		action := Action{
			player: next,
			kind:   kind,
			prob:   (float64(kind.ext) + added) / total,
		}
		actions = append(actions, action)
	}

	// 选择一个 Action
	index := -1
	if len(actions) > 0 {
		index = rand.Intn(len(actions))
	}
	return actions, 0, index
}

// 游戏模拟推演
func rollout(root, leaf *Node) float64 {
	var (
		curr     = leaf
		landlord = leaf.state.landlord
		player   = root.action.player.Next()
	)
	for !curr.state.Gameover() {
		if len(curr.children) == 0 {
			actions, _, index := getLegalActions(curr)
			for _, action := range actions {
				child := NewNode(curr, action, action.Do(curr.state))
				curr.children = append(curr.children, child)
			}
			if index < 0 || index >= len(curr.children) {
				index = rand.Intn(len(curr.children))
			}
			curr = curr.children[index]
		} else {
			totalUnvisited := 0
			for _, child := range curr.children {
				if child.n < 1 {
					totalUnvisited++
				}
			}
			if totalUnvisited == 0 {
				curr = curr.children[rand.Intn(len(curr.children))]
			} else {
				selected := rand.Intn(totalUnvisited)
				tmp := 0
				index := 0
				for i, child := range curr.children {
					if child.n < 1 {
						if tmp == selected {
							index = i
							break
						}
						tmp++
					}
				}
				curr = curr.children[index]
			}
		}
	}
	// 计算结果值
	winner := curr.state.Winner()
	multi := float64(curr.state.multi)
	if curr.state.IsSpring(winner) {
		multi *= 2
	}
	if winner.IsFriend(landlord, player) {
		return multi
	}
	return -multi
}
