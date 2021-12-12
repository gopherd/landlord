package ai

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/gopherd/log"
)

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", time.Duration(d).String())), nil
}

type runStats struct {
	NumNewNodes     int64
	NumTraverseNode int64
	TimeOfTraverse  Duration
	TimeOfExpand    Duration
	TimeOfRollout   Duration
	TimeOfBackup    Duration
}

// 蒙特卡洛搜索树(MCTS)状态节点
type Node struct {
	// 父节点
	parent *Node
	// 子节点
	children []*Node

	// 节点状态
	state State
	// 达到当前节点的动作
	action Action
	// 节点深度
	depth int

	n float64 // 节点访问次数
	q float64 // 动作奖励(action value)
	u float64 // 置信上限(upper confidence bound)
	p float64 // 先验概率(priori probability)
}

// 创建节点
// 创建根节点时, parent 和 action 为空即可
func NewNode(parent *Node, action Action, state State) *Node {
	node := &Node{
		parent: parent,
		state:  state,
		action: action,
		u:      action.prob,
		p:      action.prob,
	}
	if parent != nil {
		node.depth = parent.depth + 1
	}
	return node
}

// 输出节点信息
func (node *Node) String() string {
	var buf bytes.Buffer
	buf.WriteByte('{')
	fmt.Fprintf(&buf, "children: %d, ", len(node.children))
	fmt.Fprintf(&buf, "state: %v, ", node.state)
	fmt.Fprintf(&buf, "action: %v, ", node.action)
	fmt.Fprintf(&buf, "n: %.4g, q: %.4g, u: %.4g, p: %.4g", node.n, node.q, node.u, node.p)
	buf.WriteByte('}')
	return buf.String()
}

func (node *Node) Summary() string {
	var buf bytes.Buffer
	buf.WriteByte('{')
	fmt.Fprintf(&buf, "children: %d, ", len(node.children))
	fmt.Fprintf(&buf, "action: %v, ", node.action)
	fmt.Fprintf(&buf, "n: %.4g, q: %.4g, u: %.4g, p: %.4g", node.n, node.q, node.u, node.p)
	buf.WriteByte('}')
	return buf.String()
}

// 节点浅层拷贝
func (node *Node) clone() *Node {
	return &Node{
		parent: node.parent,
		state:  node.state,
		action: node.action,
		depth:  node.depth,
		n:      node.n,
		q:      node.q,
		u:      node.u,
		p:      node.p,
	}
}

// 执行蒙特卡洛树搜索(MCTS)
func (node *Node) Search(policyFn PolicyFunc, rolloutFn RolloutFunc, alpha, cparam float64, maxcnt int) *Node {
	// 在搜索次数和搜索时间限制下执行蒙特卡洛树搜索
	var (
		stats = runStats{}
		begin = time.Now()
		now   time.Time
	)
	for i := 0; i < maxcnt; i++ {
		// Select:
		// 从当前根节点延伸到叶子节点
		// 每次向下延伸时使用 q+u 最大的子节点
		leaf := node.traverse()

		now = time.Now()
		stats.TimeOfTraverse += Duration(now.Sub(begin))
		begin = now

		// Expand and evaluate
		var value1 float64
		leaf, value1 = leaf.expand(policyFn)

		now = time.Now()
		stats.TimeOfExpand += Duration(now.Sub(begin))
		begin = now

		var value2 float64
		if rolloutFn != nil {
			value2 = rolloutFn(node, leaf)
		}
		value := alpha*value2 + (1-alpha)*value1

		now = time.Now()
		stats.TimeOfRollout += Duration(now.Sub(begin))
		begin = now

		// Backup
		leaf.backup(node, value, cparam)

		now = time.Now()
		stats.TimeOfBackup += Duration(now.Sub(begin))
		begin = now
	}
	log.Debug().Any("stats", stats).Print("mcts Search stats")

	// 选择访问次数最多的子节点做最优解
	maxi := 0
	maxn := float64(0)
	for i, child := range node.children {
		n := child.n + rand.Float64()
		if i == 0 || n > maxn {
			maxi = i
			maxn = n
		}
	}
	return node.children[maxi]
}

// 节点推进
func (node *Node) Move(action Action) *Node {
	// 从子节点中寻找和 action 相等的子节点
	var next *Node
	for _, child := range node.children {
		if child.action.Equal(action) {
			next = child
			break
		}
	}
	// 如果没有找到则创建一个新的节点
	if next == nil {
		next = NewNode(node, action, action.Do(node.state))
	}
	// 裁剪子节点
	for _, child := range node.children {
		child.parent = nil
	}
	node.children = nil

	next.parent = node.clone()
	// 返回下一个根节点
	return next
}

// 根据 q+u 值向下遍历寻找叶子节点
func (node *Node) traverse() *Node {
	curr := node
	player := node.action.player.Next()
	for !curr.state.Gameover() {
		if len(curr.children) > 0 {
			next := curr.next(player)
			if next == nil {
				break
			}
			curr = next
		} else {
			break
		}
	}
	return curr
}

// 选取 q+u 最大的子节点
func (node *Node) next(player Position) *Node {
	var (
		maxi = -1
		maxv float64
	)
	isFriend := node.action.player.Next().IsFriend(node.state.landlord, player)
	for i, child := range node.children {
		if child.n < 1 {
			return nil
		}
		q := child.q
		u := child.u
		if !isFriend {
			q = -q
			u = -u
		}
		v := q + u
		if maxi < 0 || v > maxv {
			maxi = i
			maxv = v
		}
	}
	if maxi < 0 {
		return nil
	}
	return node.children[maxi]
}

// 使用给定策略扩展当前节点的子节点
func (node *Node) expand(policyFn PolicyFunc) (*Node, float64) {
	if len(node.children) == 0 {
		var (
			actions, value, _ = policyFn(node)
		)
		if len(actions) == 0 {
			return node, value
		}
		for _, action := range actions {
			child := NewNode(node, action, action.Do(node.state))
			node.children = append(node.children, child)
		}
		return node.children[rand.Intn(len(node.children))], value
	}

	totalUnvisited := 0
	for _, child := range node.children {
		if child.n < 1 {
			totalUnvisited++
		}
	}
	if totalUnvisited == 0 {
		return node.children[rand.Intn(len(node.children))], 0
	} else {
		selected := rand.Intn(totalUnvisited)
		tmp := 0
		index := 0
		for i, child := range node.children {
			if child.n < 1 {
				if tmp == selected {
					index = i
					break
				}
				tmp++
			}
		}
		return node.children[index], 0
	}
}

// 反向迭代更新节点统计数据(n,q,u)
func (node *Node) backup(root *Node, value, cparam float64) {
	curr := node
	for curr != nil && curr != root {
		curr.update(value, cparam, false, curr.depth-root.depth)
		curr = curr.parent
	}
	if root != nil {
		root.update(value, cparam, true, 0)
	}
}

// 更新节点统计数据(n,q,u)
func (node *Node) update(value, cparam float64, isRoot bool, depth int) {
	node.n += 1
	node.q += (value - node.q) / node.n
	if !isRoot && node.parent != nil {
		node.u = cparam * node.p * math.Sqrt(node.parent.n) / (1 + node.n)
	}
}
