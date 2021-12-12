package ai

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/gopherd/doge/graphviz"

	"github.com/gopherd/landlord/poker"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// 构建一个袖珍版斗地主游戏: 农民每人8张牌,地主10张牌
// 这个缩减版的斗地主游戏用于检查 AI 执行过程中具体所做的决策
// 执行过程会输成一个 .dot 文件，如 ``p0.dot''
// 拷贝文件内容然后在 http://viz-js.com/ 中粘贴到左侧输入框就可以可视化的查看

// TODO
// 1. mcts 搜索过程增加路径长度惩罚,这个惩罚只能占非常小的分量

func initPokers() ([NumPlayer]PokerSet, Position) {
	var pokers []poker.Poker
	for suit := poker.Suit(0); suit < 4; suit++ {
		for value := poker.P3; value <= poker.P8; value++ {
			pokers = append(pokers, poker.NewPoker(suit, value))
		}
	}
	pokers = append(pokers, poker.Joker1)
	pokers = append(pokers, poker.Joker2)

	rand.Shuffle(len(pokers), func(i, j int) {
		pokers[i], pokers[j] = pokers[j], pokers[i]
	})
	landlord := Position(rand.Intn(3))
	var ret [NumPlayer]PokerSet
	for i := 0; i < NumPlayer; i++ {
		for j := 0; j < 8; j++ {
			ret[i].Add(NewPokerSetWithPoker(pokers[len(pokers)-1]))
			pokers = pokers[:len(pokers)-1]
		}
	}
	for _, p := range pokers {
		ret[landlord.Value()].Add(NewPokerSetWithPoker(p))
	}
	return ret, landlord
}

func playout(t *testing.T, i int) {
	pokers, landlord := initPokers()
	var players [NumPlayer]*mctsAI
	for i := range players {
		players[i] = new(mctsAI)
		players[i].SetSelf(Position(i))
		players[i].SetLandlord(landlord)
		players[i].Start(pokers)
	}
	pos := landlord.Prev()
	name := fmt.Sprintf("p%d", i)
	graph := graphviz.New(name, graphviz.Directed)

	t.Logf("> Start")
	for i := range pokers {
		t.Logf("%d:\n%s", i, poker.FormatValues(pokers[i].GetValues()))
	}

	var newEntity = func(s string, n *Node, selected bool) *graphviz.Entity {
		var (
			shape = "box"
			color = "blue"
			label = ""
		)
		if n.action.player == landlord {
			color = "black"
		}
		label = fmt.Sprintf("pos: %d, node: %s\\lchildren: %d\\lN: %.4g, Q: %.4g\\lP: %.4g, U: %.4g\\lplay: %v\\lp0: %v\\lp1: %v\\lp2: %v\\l",
			n.action.player, s, len(n.children),
			n.n, n.q, n.p, n.u, n.action.kind.Pokers(),
			n.state.pokers[0].StringWithoutSuit(),
			n.state.pokers[1].StringWithoutSuit(),
			n.state.pokers[2].StringWithoutSuit(),
		)
		attr := fmt.Sprintf("[shape=%s,color=%s,label=\"%v\"]", shape, color, label)
		return graphviz.NewEntity(s, attr)
	}

	var newEdge = func(e1, e2 *graphviz.Entity, selected bool) {
		attr := ""
		if selected {
			attr = `[color=red]`
		}
		graph.Add(e1, e2, attr)
	}

	var layer = 0
	var nodeId = 0
	var nodeName = func(l, id int) string { return fmt.Sprintf("s_%d_%d", l, id) }
	lastSelectedEntity := newEntity(nodeName(layer, nodeId), players[landlord].root, true)
	for {
		layer++
		pos = pos.Next()
		tag := pos.Role(landlord)
		kind := players[pos].RecommendPlay(tag)

		from := lastSelectedEntity
		// 记录地主的博弈树
		for _, child := range players[pos].root.children {
			nodeId++
			childName := nodeName(layer, nodeId)
			selected := child.action.kind.Equal(kind)
			to := newEntity(childName, child, selected)
			newEdge(from, to, selected)
			if selected {
				lastSelectedEntity = to
			}
		}

		for i := range players {
			players[i].Play(tag, pos, kind)
		}
		t.Logf("> Play -------------------------------------------------------")
		t.Logf("> %d:\n%s", pos, poker.FormatValues(kind.Pokers().GetValues()))
		for i := range players {
			t.Logf("%d:\n%s", i, poker.FormatValues(players[i].pokers[i].GetValues()))
		}
		if players[pos].root.state.Gameover() {
			break
		}
	}
	for i := range players {
		players[i].Stop()
	}
	graph.WriteFile(name + ".dot")
}

func TestPlayout(t *testing.T) {
	const N = 1
	for i := 0; i < N; i++ {
		playout(t, i)
	}
}
