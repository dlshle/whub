package uri_trie

import (
	"errors"
	"fmt"
	"strings"
	"whub/common/utils"
)

const (
	DefaultCompactSize = 15
)

type MatchContext struct {
	UriPattern  string
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
	// stop until it hits /
	for i < len(remaining) && remaining[i] != '/' {
		i++
	}
	if i == len(remaining) {
		return remaining, ""
	}
	return remaining[:i], remaining[i:]
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
	tnTypeP  = 0
	tnTypeW  = 1
	tnTypeC  = 2
	tnTypePC = 3
	tnTypeWC = 4
)

type trieNode struct {
	parent        *trieNode
	wildcardChild *trieNode            // *
	paramChild    *trieNode            // :param
	constChildren map[string]*trieNode // const
	param         string
	value         interface{}
	path          string
	t             uint8
}

func stringifyConstChildren(node *trieNode) string {
	var builder strings.Builder
	for k := range node.constChildren {
		builder.WriteString(fmt.Sprintf("\"%s\",", k))
	}
	return builder.String()[:builder.Len()-1]
}

func (n *trieNode) addParam(param string) (*trieNode, error) {
	// we allow adding another param child w/ the same param
	if n.wildcardChild != nil || (n.paramChild != nil && n.paramChild.param != param) {
		return nil, errors.New(fmt.Sprintf("can not add a new param node \"%s\" over a wildcard/const node or a param node w/ different param \"%s\"", param, n.param))
	}
	// when overriding a child w/ value, do soft add and do not override its value
	if n.paramChild == nil {
		n.paramChild = &trieNode{parent: n, param: param, t: tnTypeP}
	}
	return n.paramChild, nil
}

func (n *trieNode) addWildcard(param string) (*trieNode, error) {
	if (n.wildcardChild != nil && n.wildcardChild.param != param) || n.paramChild != nil {
		return nil, errors.New(fmt.Sprintf("can not add a new wildcard node \"%s\" over a param/const node or a wildcard node w/ different param \"%s\"", param, n.param))
	}
	n.wildcardChild = &trieNode{parent: n, param: param, t: tnTypeW}
	return n.wildcardChild, nil
}

func (n *trieNode) addConst(subPath string) (*trieNode, error) {
	if n.constChildren == nil {
		n.constChildren = make(map[string]*trieNode)
	}
	node := n.constChildren[subPath]
	if node == nil {
		node = &trieNode{parent: n, t: tnTypeC}
		n.constChildren[subPath] = node
	}
	return node, nil
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
						errors.New(fmt.Sprintf("param %s has already been taken in url %s", param, path))
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
		case '/':
			node, err = node.addConst("/")
		default:
			// constant child
			var subPath string
			subPath, remaining = splitRemaining(remaining)
			subPath = fmt.Sprintf("%c%s", token, subPath)
			node, err = node.addConst(subPath)
		}
		if err != nil {
			return
		}
	}
	if node.value != nil && !override {
		err = errors.New(fmt.Sprintf("path %s has already been taken, please use AddPath(path, Value, true) to override current Value", path))
	} else {
		node.value = value
		node.path = path
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
		n.parent.paramChild = nil
		n.parent.wildcardChild = nil
		if n.parent.constChildren != nil {
			for k, v := range n.parent.constChildren {
				if v == n {
					n.parent.constChildren[k] = nil
				}
			}
		}
	}
	n.parent = nil
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

func (n *trieNode) findByPath(path string) *trieNode {
	if len(path) == 0 {
		return nil
	}
	curr := n
	remaining := path
	for len(remaining) > 0 {
		if curr == nil {
			break
		}
		token := remaining[0]
		remaining = remaining[1:]
		switch token {
		case '/':
			curr = curr.constChildren["/"]
		default:
			var subPath string
			subPath, remaining = splitRemaining(remaining)
			// Match const first. When paths like /a/:x and /a/b both exist, we need to match const path first and then param/wildcard.
			if tCurr := curr.constChildren[fmt.Sprintf("%c%s", token, subPath)]; tCurr != nil {
				curr = tCurr
				continue
			}
			if curr.wildcardChild != nil {
				curr = curr.wildcardChild
				break
			} else if curr.paramChild != nil {
				curr = curr.paramChild
			}
		}
	}
	return curr
}

func (n *trieNode) match(path string, ctx *MatchContext) (node *trieNode, err error) {
	if len(path) == 0 {
		return nil, errors.New("no path find")
	}
	curr := n
	remaining := path
	for len(remaining) > 0 {
		if curr == nil {
			err = errors.New(fmt.Sprintf("mismatched remaining path %s from %s- no routing found", remaining, path))
			curr = nil
			break
		}
		token := remaining[0]
		remaining = remaining[1:]
		switch token {
		case '/':
			curr = curr.constChildren["/"]
		default:
			var subPath string
			subPath, remaining = splitRemaining(remaining)
			subPath = fmt.Sprintf("%c%s", token, subPath)
			// Match const first. When paths like /a/:x and /a/b both exist, we need to match const path first and then param/wildcard.
			if tCurr := curr.constChildren[subPath]; tCurr != nil {
				curr = tCurr
				continue
			}
			if curr.wildcardChild != nil {
				// add param
				ctx.PathParams[curr.wildcardChild.param] = fmt.Sprintf("%s%s", subPath, remaining)
				curr = curr.wildcardChild
				break
			} else if curr.paramChild != nil {
				// add param
				ctx.PathParams[curr.paramChild.param] = subPath
				curr = curr.paramChild
			}
		}
	}
	if curr != nil {
		node = curr
		ctx.Value = node.value
		ctx.UriPattern = node.path
	} else if err == nil {
		err = errors.New(fmt.Sprintf("no routing found for path %s", path))
	}
	return
}

func (n *trieNode) matchByPath(pathWithoutQueryParams string, ctx *MatchContext) (c *MatchContext, err error) {
	if len(pathWithoutQueryParams) == 0 {
		return nil, errors.New("no path find")
	}
	node, err := n.match(pathWithoutQueryParams, ctx)
	if err != nil || node == nil {
		return
	}
	if node.value == nil {
		return nil, errors.New(fmt.Sprintf("no value associated with path %s", pathWithoutQueryParams))
	}
	c = ctx
	return
}

func (n *trieNode) r_path() (path string, isConst bool) {
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
	root *trieNode
	size int
}

func NewTrieTree() *TrieTree {
	return &TrieTree{
		root: &trieNode{parent: nil},
	}
}

func (t *TrieTree) Size() int {
	return t.size
}

func (t *TrieTree) Match(path string) (*MatchContext, error) {
	if path == "" {
		return nil, errors.New("empty path")
	}
	path = t.sanitizePath(path)
	paramStr, remaining := splitQueryParams(path)
	queryParams, err := parseQueryParams(paramStr)
	if err != nil {
		return nil, err
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
	_, err := t.root.addPath(UriContext{make(map[string]bool)}, path, value, override)
	t.size++
	return err
}

func (t *TrieTree) Remove(path string) bool {
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
	path = t.sanitizePath(path)
	paramStr, remaining := splitQueryParams(path)
	_, err := parseQueryParams(paramStr)
	if err != nil {
		return false
	}
	n := t.root.findByPath(remaining)
	if n == nil {
		return false
	}
	return n.value != nil
}

func (t *TrieTree) RemoveAll() {
	t.root.clean()
}

func (t *TrieTree) sanitizePath(path string) string {
	// special case when there's an extra '/' at the bottom of path
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return path
}