package uri_trie

import (
	"testing"
	"wsdk/common/test_utils"
)

func TestTrieTree(t *testing.T) {
	tree := NewTrieTree()
	test_utils.NewTestGroup("trie tree", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("Add wildcard", "", func() bool {
			err := tree.Add("/x/*z", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("Match wildcard", "", func() bool {
			return tree.SupportsUri("/x/asd")
		}),
		test_utils.NewTestCase("Add const", "", func() bool {
			err := tree.Add("/x/z", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("Match const", "", func() bool {
			return tree.SupportsUri("/x/z")
		}),
		test_utils.NewTestCase("Add param", "", func() bool {
			// TODO err here!
			err := tree.Add("/x/z/:p/x", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("Match param", "", func() bool {
			ctx, err := tree.Match("/x/z/param/x")
			if err != nil {
				return false
			}
			return ctx.PathParams["p"] == "param"
		}),
		test_utils.NewTestCase("Add short wildcard", "", func() bool {
			tree.RemoveAll()
			// TODO err here!
			err := tree.Add("/*z", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("Match short wildcard", "", func() bool {
			ctx, err := tree.Match("/xyz")
			if err != nil {
				return false
			}
			return ctx.PathParams["z"] == "xyz"
		}),
	}).Do(t)
}
