package internal

import "sort"

type Cursor struct {
	tree    *Tree
	leaf    *Page
	idx     int
	start   []byte
	end     []byte
	started bool
}

func (t *Tree) NewCursor(start, end []byte) *Cursor {
	return &Cursor{tree: t, start: start, end: end}
}

// Next returns the next KV in range, advancing internal position.
func (c *Cursor) Next() (key, value []byte, ok bool, err error) {
	if !c.started {
		c.leaf, _, err = c.tree.findLeaf(c.start)
		if err != nil {
			return nil, nil, false, err
		}
		c.idx = sort.Search(len(c.leaf.Keys), func(i int) bool {
			return KeyCompare(c.leaf.Keys[i], c.start) >= 0
		})
		c.started = true
	}

	for {
		if c.leaf == nil {
			return nil, nil, false, nil
		}
		if c.idx >= len(c.leaf.Keys) {
			if c.leaf.NextLeaf == 0 {
				c.leaf = nil
				return nil, nil, false, nil
			}
			c.leaf, err = c.tree.loadPage(c.leaf.NextLeaf)
			if err != nil {
				return nil, nil, false, err
			}
			c.idx = 0
			continue
		}

		k := c.leaf.Keys[c.idx]
		if KeyCompare(k, c.end) > 0 {
			c.leaf = nil // past range, done
			return nil, nil, false, nil
		}

		v := c.leaf.Values[c.idx]
		c.idx++
		return k, v, true, nil
	}
}
