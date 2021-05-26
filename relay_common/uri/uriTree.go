package uri

import (
	"errors"
	"strings"
)

func isWildcard(v string) bool {
	return v[0] == ':'
}

func getParameters(path string) (map[string]string, error){
	i := strings.LastIndex(path, "?")
	l := len(path)
	if i == -1 || i == l - 1 {
		return nil, errors.New("invalid parameter: incorrect ? position")
	}
	params := strings.Split(path[i+1:], "&")
	paramsMap := make(map[string]string)
	for _, param := range params {
		sep := strings.Index(param, "=")
		if sep == -1 || sep == len(param) - 1 {
			return nil, errors.New("invalid parameter: incorrect = position")
		}
		paramsMap[param[:sep]] = param[sep+1:]
	}
	return paramsMap, nil
}

type UriMatchResult struct {
	isMatch bool
	pathParams map[string]string
	params map[string]string
}

func NewUriMatchResult() *UriMatchResult {
	return &UriMatchResult{
		isMatch: false,
		pathParams: make(map[string]string),
	}
}

// * can only be placed at the bottom
type UriNode struct {
	placeHolder string // could be :something for tagged place holder or "" for none.
	parent *UriNode
	leaves map[string]*UriNode
}

func NewNode() *UriNode {
	return &UriNode{}
}

func NewWildcardNode(placeHolder string) *UriNode {
	return &UriNode{placeHolder: placeHolder, leaves: make(map[string]*UriNode)}
}

func (n *UriNode) Value() string {
	if n.parent == nil {
		return ""
	}
	for k, v := range n.parent.leaves {
		if v == n {
			return k
		}
	}
	return ""
}

func (n *UriNode) PlaceHolder() string {
	return n.placeHolder
}

// very costly, try not to use
func (n *UriNode) Path() string {
	if n.parent == nil {
		return ""
	}
	return n.parent.Value() + n.Value()
}

func (n *UriNode) MatchPath(path string) *UriMatchResult {
	segs := strings.Split(path, "/")
	if len(segs) == 0 {
		return nil
	}
	res := NewUriMatchResult()
	l := len(segs)
	curr := n
	for i := 0; i < l; i++ {
		if curr.leaves[segs[i]] != nil {
			curr = curr.leaves[segs[i]]
		}
		if curr.leaves["*"] != nil {
			curr = curr.leaves["*"]
			res.pathParams[curr.placeHolder] = segs[i]
		}
		return res
	}
	if len(curr.leaves) > 0 {
		// not a leaf
		return res
	}
	// params check
	params, err := getParameters(path)
	if err != nil {
		return nil
	}
	res.params = params
	return res
}

// returns newly added node or nil
func (n *UriNode) AddSeg(seg string) *UriNode {
	isWC := isWildcard(seg)
	if n.leaves[seg] != nil {
		if isWC {
			// duplicate wildcard uri
			return nil
		}
		return n.leaves[seg]
	}
	var node *UriNode
	if isWC {
		node = NewWildcardNode(seg)
	} else {
		node = NewNode()
	}
	node.parent = n
	return node
}

// returns the leaf node or nil
func (n *UriNode) AddPath(path string) *UriNode {
	segs := strings.Split(path, "/")
	if len(segs) < 1 {
		return nil
	}
	l := len(segs)
	curr := n
	for i := 0; i < l; i++ {
		curr = curr.AddSeg(segs[i])
		if curr == nil {
			return nil
		}
	}
	return curr
}

func (n *UriNode) Remove() {
	if n.parent == nil {
		return
	}
	if len(n.leaves) > 0 {
		// can not disconnect the non-leaf nodes on removal
		n.parent.Remove()
		return
	}
	for k, v := range n.parent.leaves {
		if v == n {
			delete(n.parent.leaves, k)
		}
	}
	n.parent.Remove()
}

