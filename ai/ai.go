package ai

import (
	"fmt"
	"math/rand"

	"github.com/gopherd/log"
)

// 一个 3 人斗地主的 AI
type AI interface {
	// 设置地主位置
	SetLandlord(Position)
	// 设置底牌
	SetLastPokers(PokerSet)
	// 设置自己的位置
	SetSelf(Position)
	// 玩家叫地主
	Rob(Position, int)
	// 玩家加倍
	Double(Position, int)
	// 玩家出牌
	Play(tag string, pos Position, kind Kind)
	// 建议叫地主
	RecommendRob() int
	// 建议加倍
	RecommendDouble() int
	// 建议出牌
	RecommendPlay(tag string) Kind
	// 准备开始出牌了
	Start(pokers [NumPlayer]PokerSet)
	// 结束
	Stop()
}

// 实现一个类似蒙特卡罗树搜索(MCTS)算法的 AI
type mctsAI struct {
	// 地主位置
	landlord Position
	// 底牌
	lastPokers PokerSet
	// 自己的位置
	self Position
	// 叫地主分数
	scores [NumPlayer]int
	// 加倍倍数
	multiples [NumPlayer]int
	// 各玩家剩余牌
	pokers [NumPlayer]PokerSet
	// 当前状态
	root *Node
}

func (ai *mctsAI) SetLandlord(pos Position)      { ai.landlord = pos }
func (ai *mctsAI) SetLastPokers(pokers PokerSet) { ai.lastPokers = pokers }
func (ai *mctsAI) SetSelf(pos Position)          { ai.self = pos }

func (ai *mctsAI) Rob(pos Position, score int)    { ai.scores[pos.Value()] = score }
func (ai *mctsAI) Double(pos Position, multi int) { ai.multiples[pos.Value()] = multi }

func (ai *mctsAI) Start(pokers [NumPlayer]PokerSet) {
	copy(ai.pokers[:], pokers[:])
	ai.root = new(Node)
	ai.root.state = NewState(pokers, ai.landlord)
	ai.root.action.player = ai.landlord.Prev()
}

func (ai *mctsAI) Stop() {
	ai.root = nil
}

func (ai *mctsAI) next() Position {
	if ai.root.parent == nil {
		return ai.landlord
	}
	return ai.root.action.player.Next()
}

// 记录出牌结果,推进游戏状态
func (ai *mctsAI) Play(tag string, pos Position, kind Kind) {
	log.Debug().Any("pos", pos).Any("kind", kind).Print("mctsAI Play")
	next := ai.next()
	if next != pos {
		panic(fmt.Sprintf("next position should be %v, but got %v", next, pos))
	}
	// 推进搜索树节点
	ai.root = ai.root.Move(Action{
		player: pos,
		kind:   kind,
	})
	ai.pokers[pos].Remove(kind.Pokers())
}

// 建议叫地主分数, 0 表示不叫
func (ai *mctsAI) RecommendRob() int {
	return rand.Intn(4)
}

// 建议加倍倍数, 0 表示不加倍, 2 表示加倍
func (ai *mctsAI) RecommendDouble() int {
	return rand.Intn(2) * 2
}

// 建议出牌
func (ai *mctsAI) RecommendPlay(tag string) Kind {
	log.Debug().Any("current", ai.root).Print("mctsAI RecommendPlay")
	const c = 30
	var maxcnt int
	numPokers := ai.root.state.NumPokers()
	maxcnt = numPokers*numPokers*2 + 100
	node := ai.root.Search(getLegalActions, rollout, 1, c, maxcnt)
	if node == nil {
		panic("selected node is nil")
	}
	log.Debug().Any("node", node).Print("mctsAI RecommendPlay")

	var (
		pos    = node.action.player
		kind   = node.action.kind
		pokers = ai.pokers[pos]
	)
	body := pokers.Find(kind.body)
	pokers.Remove(body)
	kicker := pokers.Find(kind.kicker)
	return kind.extend(body, kicker)
}
