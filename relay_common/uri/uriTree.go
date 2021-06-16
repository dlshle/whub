package uri

import (
	"errors"
	"fmt"
	"strings"
	"wsdk/gommon/data_structures"
	"wsdk/relay_common/utils"
)

const DefaultCompactSize = 16

func parseQueryParams(queryParamString string) (pMap map[string]string, err error) {
	//xx=1&&yy=2...
	pMap = make(map[string]string)
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
	i := 0
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

type UriContext struct {
	params map[string]bool
}

const (
	nTypeP = 0
	nTypeW = 1
	nTypeC = 2
)

type uriNode struct {
	parent        *uriNode
	wildcardChild *uriNode            // *
	paramChild    *uriNode            // :param
	constChildren map[string]*uriNode // const
	param         string
	handler       func(pathParams map[string]string, queryParams map[string]string) error
	t             uint8
}

func (n *uriNode) addParam(param string) (*uriNode, error) {
	if n.paramChild != nil {
		return nil, errors.New("no duplicated param child for single node")
	}
	n.paramChild = &uriNode{parent: n, param: param, t: nTypeP}
	return n.paramChild, nil
}

func (n *uriNode) addWildcard(param string) (*uriNode, error) {
	if n.wildcardChild != nil {
		return nil, errors.New("no duplicate wildcard child for single node")
	}
	n.wildcardChild = &uriNode{parent: n, param: param, t: nTypeW}
	return n.wildcardChild, nil
}

func (n *uriNode) addConst(subPath string) *uriNode {
	if n.constChildren == nil {
		n.constChildren = make(map[string]*uriNode)
	}
	var node *uriNode
	node = n.constChildren[subPath]
	if node == nil {
		node = &uriNode{parent: n, t: nTypeC}
		n.constChildren[subPath] = node
	}
	return node
}

func (n *uriNode) addPath(ctx UriContext, path string, handler func(map[string]string, map[string]string) error, override bool) (node *uriNode, err error) {
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
	if node.handler != nil && !override {
		err = errors.New(fmt.Sprintf("path %s has already been taken, please use AddPath(path, handler, true) to override current handler", path))
	} else {
		node.handler = handler
	}
	return
}

func (n *uriNode) remove() {
	if n.parent == nil {
		return
	}
	if !(n.paramChild == nil || n.wildcardChild == nil || n.constChildren == nil || len(n.constChildren) == 0) {
		return
	} else {
		// safe to remove, remove current node from its parent
		switch n.t {
		case nTypeP:
			n.parent.paramChild = nil
		case nTypeW:
			n.parent.wildcardChild = nil
		case nTypeC:
			for k, v := range n.parent.constChildren {
				if v == n {
					n.parent.constChildren[k] = nil
				}
			}
		}
		n.parent = nil
	}
}

func (n *uriNode) findWithoutQueryParams(path string) (node *uriNode, params map[string]string, err error) {
	if len(path) == 0 {
		return nil, nil, errors.New("no path find")
	}
	params = make(map[string]string)
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
			params[curr.param] = subPath
			break
		} else if curr.paramChild != nil {
			curr = curr.paramChild
			params[curr.param] = subPath
		} else {
			err = errors.New(fmt.Sprintf("mismatch subpath %s from %s- not routing found", subPath, path))
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

func (n *uriNode) getHandler(pathWithoutQueryParams string, queryParams map[string]string) (handler func() error, err error) {
	if len(pathWithoutQueryParams) == 0 {
		return nil, errors.New("no pathWithoutQueryParams find")
	}
	var params map[string]string
	if err != nil {
		return nil, err
	}
	node, params, err := n.findWithoutQueryParams(pathWithoutQueryParams)
	handler = func() error { return node.handler(params, queryParams) }
	return
}

func (n *uriNode) path() (path string, isConst bool) {
	curr := n
	isConst = true
	for curr != nil {
		if n.t != nTypeC {
			isConst = false
		}
		if curr.param != "" {
			if curr != n {
				path = curr.param + "/" + path
			} else {
				path = curr.param
			}
		} else if curr.parent != nil {
			for k, v := range n.parent.constChildren {
				if v == n {
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

type UriTree struct {
	root              *uriNode
	constPathMap      map[string]func(queryParams map[string]string) error // initially nil, when a new path has no : or *, it will be registered
	unCompactedLeaves *data_structures.LinkedList
	uriContext        UriContext
	size              int
}

func NewUriTree() *UriTree {
	return &UriTree{
		root:              &uriNode{parent: nil},
		constPathMap:      make(map[string]func(map[string]string) error),
		unCompactedLeaves: data_structures.NewLinkedList(false),
		uriContext:        UriContext{params: make(map[string]bool)},
	}
}

func (t *UriTree) Size() int {
	return t.size
}

func (t *UriTree) compact() {
	for t.unCompactedLeaves.Size() > 0 {
		node := t.unCompactedLeaves.Pop().(*uriNode)
		path, isConst := node.path()
		if isConst {
			node.remove()
			t.constPathMap[path] = func(queryParams map[string]string) error {
				return node.handler(make(map[string]string), queryParams)
			}
		}
	}
}

func (t *UriTree) FindAndHandle(path string) error {
	if path == "" {
		return errors.New("empty path")
	}
	paramStr, remaining := splitQueryParams(path)
	qp, err := parseQueryParams(paramStr)
	if err != nil {
		return err
	}
	if t.constPathMap[remaining] != nil {
		if err != nil {
			return err
		}
		return t.constPathMap[remaining](qp)
	}
	h, e := t.root.getHandler(remaining, qp)
	if h == nil || e != nil {
		return e
	}
	return h()
}

func (t *UriTree) Add(path string, handler func(map[string]string, map[string]string) error, override bool) error {
	node, err := t.root.addPath(t.uriContext, path, handler, override)
	t.unCompactedLeaves.Append(node)
	if t.unCompactedLeaves.Size() >= DefaultCompactSize {
		t.compact()
	}
	t.size++
	return err
}

func (t *UriTree) Remove(path string) error {
	if t.constPathMap[path] != nil {
		delete(t.constPathMap, path)
		t.size--
		return nil
	}
	node, _, err := t.root.findWithoutQueryParams(path)
	if err != nil {
		return err
	}
	node.remove()
	t.size--
	return nil
}

func (t *UriTree) SupportsUri(path string) bool {
	if path == "" {
		return false
	}
	paramStr, remaining := splitQueryParams(path)
	qp, err := parseQueryParams(paramStr)
	if err != nil {
		return false
	}
	if t.constPathMap[remaining] != nil {
		if err != nil {
			return false
		}
		return true
	}
	h, e := t.root.getHandler(remaining, qp)
	if h == nil || e != nil {
		return false
	}
	return true
}
