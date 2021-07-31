package uri_trie

import (
	"errors"
	"fmt"
	"strings"
	"wsdk/common/data_structures"
	"wsdk/relay_common/utils"
)

const (
	DefaultCompactSize = 15
)

type MatchContext struct {
	QueryParams map[string]string
	PathParams  map[string]string
	Value       interface{}
}

type UriContext struct {
	params map[string]bool
}

func parseQueryParams(queryParamString string) (pMap map[string]string, err error) {
	//xx=1&&yy=2...
	pMap = make(map[string]string)
	if queryParamString == "" {
		return pMap, nil
	}
	exps := strings.Split(queryParamString, "&")
	for _, exp := range exps {
		split := strings.Split(exp, "=")
		if len(split) != 2 {
			err = errors.New("invalid expression " + exp)
			pMap = nil
			return
		}
		pMap[split[0]] = split[1]
	}
	return
}

func splitRemaining(remaining string) (string, string) {
	if len(remaining) == 0 {
		return "", ""
	}
	i := 0
	// special case when url starts with '/' (e.g. /service_manager/xx/yy)
	if remaining[0] == '/' {
		i = 1
	}
	// stop until it hits /
	for i < len(remaining) && remaining[i] != '/' {
		i++
	}
	if i == len(remaining) {
		return remaining, ""
	}
	return remaining[:i], remaining[i+1:]
}

func splitQueryParams(path string) (queries string, remaining string) {
	remaining = path
	iSplitter := strings.LastIndex(path, "?")
	if iSplitter == -1 {
		return
	}
	return path[iSplitter+1:], path[0:iSplitter]
}

const (
	tnTypeP = 0
	tnTypeW = 1
	tnTypeC = 2
)

type trieNode struct {
	parent        *trieNode
	wildcardChild *trieNode            // *
	paramChild    *trieNode            // :param
	constChildren map[string]*trieNode // const
	param         string
	value         interface{}
	t             uint8
}

func (n *trieNode) addParam(param string) (*trieNode, error) {
	if n.paramChild != nil {
		return nil, errors.New("no duplicated param child for single node")
	}
	n.paramChild = &trieNode{parent: n, param: param, t: tnTypeP}
	return n.paramChild, nil
}

func (n *trieNode) addWildcard(param string) (*trieNode, error) {
	if n.wildcardChild != nil {
		return nil, errors.New("no duplicate wildcard child for single node")
	}
	n.wildcardChild = &trieNode{parent: n, param: param, t: tnTypeW}
	return n.wildcardChild, nil
}

func (n *trieNode) addConst(subPath string) *trieNode {
	if n.constChildren == nil {
		n.constChildren = make(map[string]*trieNode)
	}
	var node *trieNode
	node = n.constChildren[subPath]
	if node == nil {
		node = &trieNode{parent: n, t: tnTypeC}
		n.constChildren[subPath] = node
	}
	return node
}

func (n *trieNode) addPath(ctx UriContext, path string, value interface{}, override bool) (node *trieNode, err error) {
	if len(path) == 0 {
		return
	}
	node = n
	remaining := path
	for len(remaining) > 0 {
		token := remaining[0]
		remaining = remaining[1:]
		switch token {
		case ':':
			// param child
			var param string
			param, remaining = splitRemaining(remaining)
			err = utils.ProcessWithErrors(
				func() error {
					if ctx.params[param] {
						errors.New(fmt.Sprintf("param %s has already been taken", param))
					}
					return nil
				},
				func() error {
					node, err = node.addParam(param)
					return err
				},
			)
		case '*':
			// wildcard child(with param)
			param := remaining
			remaining = ""
			node, err = node.addWildcard(param)
		default:
			// constant child
			var subPath string
			subPath, remaining = splitRemaining(remaining)
			subPath = fmt.Sprintf("%c%s", token, subPath)
			node = node.addConst(subPath)
		}
		if err != nil {
			return
		}
	}
	if node.value != nil && !override {
		err = errors.New(fmt.Sprintf("path %s has already been taken, please use AddPath(path, Value, true) to override current Value", path))
	} else {
		node.value = value
	}
	return
}

func (n *trieNode) remove() {
	if n.parent == nil {
		return
	}
	if !(n.paramChild == nil || n.wildcardChild == nil || n.constChildren == nil || len(n.constChildren) == 0) {
		return
	} else {
		// safe to remove, remove current node from its parent
		switch n.t {
		case tnTypeP:
			n.parent.paramChild = nil
		case tnTypeW:
			n.parent.wildcardChild = nil
		case tnTypeC:
			for k, v := range n.parent.constChildren {
				if v == n {
					n.parent.constChildren[k] = nil
				}
			}
		}
		n.parent = nil
	}
}

// clean from up to bottom
func (n *trieNode) clean() {
	if n.paramChild != nil {
		n.paramChild.clean()
	}
	if n.wildcardChild != nil {
		n.wildcardChild.clean()
	}
	for k, c := range n.constChildren {
		c.clean()
		delete(n.constChildren, k)
	}
	n.value = nil
}

func (n *trieNode) findByPath(path string) (node *trieNode) {
	if len(path) == 0 {
		return nil
	}
	curr := n
	remaining := path
	for len(remaining) > 0 {
		var subPath string
		subPath, remaining = splitRemaining(remaining)
		// need to match constChildren first so that one subPath can be either const or param
		if curr.constChildren[subPath] != nil {
			curr = curr.constChildren[subPath]
		} else if curr.wildcardChild != nil {
			curr = curr.paramChild
			break
		} else if curr.paramChild != nil {
			curr = curr.paramChild
		} else {
			curr = nil
			break
		}
	}
	if curr != nil {
		node = curr
	}
	return
}

func (n *trieNode) match(path string, ctx *MatchContext) (node *trieNode, err error) {
	if len(path) == 0 {
		return nil, errors.New("no path find")
	}
	curr := n
	remaining := path
	for len(remaining) > 0 {
		var subPath string
		subPath, remaining = splitRemaining(remaining)
		// need to match constChildren first so that one subPath can be either const or param
		if curr.constChildren[subPath] != nil {
			curr = curr.constChildren[subPath]
		} else if curr.wildcardChild != nil {
			curr = curr.paramChild
			ctx.PathParams[curr.param] = subPath
			break
		} else if curr.paramChild != nil {
			curr = curr.paramChild
			ctx.PathParams[curr.param] = subPath
		} else {
			err = errors.New(fmt.Sprintf("mismatch subpath %s from %s- no routing found", subPath, path))
			curr = nil
			break
		}
	}
	if curr != nil {
		node = curr
	} else if err == nil {
		err = errors.New(fmt.Sprintf("no routing found for path %s", path))
	}
	return
}

func (n *trieNode) matchByPath(pathWithoutQueryParams string, ctx *MatchContext) (c *MatchContext, err error) {
	if len(pathWithoutQueryParams) == 0 {
		return nil, errors.New("no pathWithoutQueryParams find")
	}
	node, err := n.match(pathWithoutQueryParams, ctx)
	if err != nil || node == nil {
		return
	}
	ctx.Value = node.value
	c = ctx
	return
}

func (n *trieNode) path() (path string, isConst bool) {
	curr := n
	isConst = true
	for curr != nil {
		if n.t != tnTypeC {
			isConst = false
		}
		if curr.param != "" {
			if curr != n {
				path = curr.param + "/" + path
			} else {
				path = curr.param
			}
		} else if curr.parent != nil {
			for k, v := range curr.parent.constChildren {
				if v == curr {
					if curr != n {
						path = k + "/" + path
					} else {
						path = k
					}
				}
			}
		}
		curr = curr.parent
	}
	return
}

type TrieTree struct {
	root              *trieNode
	constPathMap      map[string]interface{} // initially nil, when a new path has no : or *, it will be registered
	unCompactedLeaves *data_structures.LinkedList
	uriContext        UriContext
	size              int
}

func NewTrieTree() *TrieTree {
	return &TrieTree{
		root:              &trieNode{parent: nil},
		constPathMap:      make(map[string]interface{}),
		unCompactedLeaves: data_structures.NewLinkedList(false),
		uriContext:        UriContext{params: make(map[string]bool)},
	}
}

func (t *TrieTree) Size() int {
	return t.size
}

func (t *TrieTree) compact() {
	for t.unCompactedLeaves.Size() > 0 {
		node := t.unCompactedLeaves.Pop().(*trieNode)
		path, isConst := node.path()
		if isConst {
			node.remove()
			t.constPathMap[path] = node.value
		}
	}
}

func (t *TrieTree) Match(path string) (*MatchContext, error) {
	if path == "" {
		return nil, errors.New("empty path")
	}
	paramStr, remaining := splitQueryParams(path)
	queryParams, err := parseQueryParams(paramStr)
	if err != nil {
		return nil, err
	}
	if t.constPathMap[remaining] != nil {
		if err != nil {
			return nil, err
		}
		return &MatchContext{
			QueryParams: queryParams,
			Value:       t.constPathMap[remaining],
		}, nil
	}
	c, e := t.root.matchByPath(remaining, &MatchContext{
		PathParams:  make(map[string]string),
		QueryParams: queryParams,
	})
	if c == nil || e != nil {
		return nil, e
	}
	return c, nil
}

func (t *TrieTree) Add(path string, value interface{}, override bool) error {
	node, err := t.root.addPath(t.uriContext, path, value, override)
	t.unCompactedLeaves.Append(node)
	if t.unCompactedLeaves.Size() >= DefaultCompactSize {
		t.compact()
	}
	t.size++
	return err
}

func (t *TrieTree) Remove(path string) bool {
	if t.constPathMap[path] != nil {
		delete(t.constPathMap, path)
		t.size--
		return true
	}
	node := t.root.findByPath(path)
	if node == nil {
		return false
	}
	node.remove()
	t.size--
	return true
}

func (t *TrieTree) SupportsUri(path string) bool {
	if path == "" {
		return false
	}
	paramStr, remaining := splitQueryParams(path)
	_, err := parseQueryParams(paramStr)
	if err != nil {
		return false
	}
	if t.constPathMap[remaining] != nil {
		if err != nil {
			return false
		}
		return true
	}
	n := t.root.findByPath(remaining)
	if n == nil {
		return false
	}
	return true
}

func (t *TrieTree) RemoveAll() {
	for k, _ := range t.constPathMap {
		delete(t.constPathMap, k)
	}
	t.root.clean()
}
