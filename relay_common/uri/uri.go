package uri

import (
	"strings"
)

/*
 * Uri Format:
 * /root_uri_type/x/y/z
 * e.g. Service Uri Format:
 * /SERVICE_PREFIX/SERVICE_ID/x/{place_holder1}/y/{place_holder2}/leaf?[param1][param2]
 * wildcards:
 * node value can have wildcards at the bottom for params
 */

func isConstNodeValue(value string) bool {
	l := len(value)
	return value[l-1] != '*' && !(value[0] == '{' || value[0] == '}')
}

func getFirstNodeValue(path string) string {
	i := strings.IndexByte(path, '/')
	if i < 0 {
		return ""
	}
	return path[:i]
}

type uriNode struct {
	// ROOT node should always be {root}
	parentNode         *uriNode
	value              string // if a path has a node with {xyz}, then if another path at the same node uses wildcard would cause error!
	leaves             map[string]*uriNode
	hasWildcardLeaf    bool
	// TODO: add depth for node???
}

func newServiceUriNode(parentNode *uriNode, value string) *uriNode {
	return &uriNode{
		parentNode: parentNode,
		value: value,
		leaves: make(map[string]*uriNode),
	}
}

// TODO should really allow adding a leaf with children???
func (n *uriNode) addLeaf(leaf *uriNode) bool {
	isConstNode := isConstNodeValue(leaf.value)
	// no repeat node, no duplicate wildcard leaf
	if n.leaves[leaf.value] != nil || !isConstNode && n.hasWildcardLeaf {
		return false
	}
	if isConstNode {
		n.hasWildcardLeaf = true
	}
	n.leaves[leaf.value] = leaf
	return true
}

func (n *uriNode) isLeaf() bool {
	return len(n.leaves) == 0
}

func (n *uriNode) isPathMatch(path string) bool {

}

func (n *uriNode) isNodeMatch(target string) bool {
	nodeVal := n.value
	l := len(nodeVal)
	if nodeVal[0] == '{' && nodeVal[l-1] == '}' {
		// {xxx} matches anything
		return true
	} else if len(target) > l && nodeVal[l-1] == '*' && nodeVal[:l-1] == target[:l-1] {
		// xxx* matches xxx*****
		return true
	}
	return false
}

func (n *uriNode) path() string {
	if n.parentNode == nil {
		return n.value
	}
	return n.parentNode.value + n.value
}

func (n *uriNode) rm() {
	if len(n.leaves) > 0 || n.parentNode == nil {
		return
	}
	delete(n.parentNode.leaves, n.value)
	n.parentNode.rm()
}

type UriManager struct {
	root    *uriNode
	numUris int
}

func NewUriManager(uri string) *UriManager {
	root := transformStringToUriNodes(uri)
	if root == nil {
		return nil
	}
	return &UriManager{
		root:    root,
		numUris: 1,
	}
}

type IServiceUriManager interface {
	SupportsUri(fullUri string) bool
	Register(fullUri string) bool
	Unregister(fullUri string) bool
}

// fullUri should start with /
func transformStringToUriNodes(fullUri string) *uriNode {
	uriNodeStrs := strings.Split(fullUri, "/")
	l := len(uriNodeStrs)
	if l < 2 {
		return nil
	}
	var root, leaf *uriNode
	root = newServiceUriNode(nil, uriNodeStrs[0])
	leaf = root
	for i := 1; i < l; i++ {
		next := newServiceUriNode(leaf, uriNodeStrs[i])
		leaf.addLeaf(next)
		leaf = next
	}
	return root
}

func matchWithUri(nodeVal string, target string) bool {
	if nodeVal == target {
		return true
	}
	l := len(nodeVal)
	if nodeVal[0] == '{' && nodeVal[l-1] == '}' {
		// {xxx} matches anything
		return true
	} else if len(target) > l && nodeVal[l-1] == '*' && nodeVal[:l-1] == target[:l-1] {
		// xxx* matches xxx*****
		return true
	}
	return false
}

func (m *UriManager) SupportsUri(fullUri string) bool {
	segments := strings.Split(fullUri, "/")
	// seg[0] == SERVICE_PREFIX
	if len(segments) < 2 {
		return false
	}
	// TODO traverse each segment and if not match return false
	root := m.root
	for _, segment := range segments {
		if !matchWithUri(root.value, segment)  {
			return false
		}
		if root.leaves[segment] != nil {
			// TODO how??
		}
	}
}