package internal

import (
	"errors"
	"sort"
)

const maxChildrenPerInternalNode = 4

type Tree struct {
	pager  Pager
	rootID PageID
}

func NewBtree(p Pager, rootPageID PageID) *Tree {
	return &Tree{pager: p, rootID: rootPageID}
}

func (t *Tree) RootID() PageID { return t.rootID }

func (t *Tree) loadPage(id PageID) (*Page, error) {
	page, err := t.pager.GetPage(id)

	if errors.Is(err, ErrEmptyPage) {
		return &Page{
			ID:      id,
			IsLeaf:  true,
			NumKeys: 0,
		}, nil
	}

	if err != nil {
		return nil, err
	}

	// TODO
	// if id == core.HeaderPageID {
	// 	_, data = splitHeaderPageData(data)
	// }

	// page, err := Decode(data)

	// if err != nil {
	// 	return nil, err
	// }

	// if page.IsLeaf && len(page.Keys) != len(page.Values) {
	// 	return nil, fmt.Errorf("btree: load page %d: has %v keys but %v values", id, len(page.Keys), len(page.Values))
	// }

	// page.ID = id

	return page, nil
}

// func splitHeaderPageData(data []byte) ([]byte, []byte) {
// 	if len(data) < core.MetadataLength {
// 		return nil, data
// 	}

// 	return data[:core.MetadataLength], data[core.MetadataLength:]
// }

// func joinHeaderPageData(data []byte) []byte {
// 	metadata := core.EncodeMetadata()

// 	result := make([]byte, len(metadata)+len(data))
// 	copy(result, metadata)
// 	copy(result[len(metadata):], data)
// 	return result
// }

func (t *Tree) savePage(p *Page) error {
	// data, err := p.Encode()
	// if err != nil {
	// 	return err
	// }

	// if p.ID == core.HeaderPageID {
	// 	data = joinHeaderPageData(data)
	// }

	return t.pager.PutPage(p.ID, p)
}

type pathEntry struct {
	pageID PageID
	index  int
}

func (t *Tree) findLeaf(key []byte) (*Page, []pathEntry, error) {
	var path []pathEntry
	currentID := t.rootID

	for {
		page, err := t.loadPage(currentID)

		if err != nil {
			return nil, nil, err
		}

		if page.IsLeaf {
			return page, path, nil
		}

		i := sort.Search(len(page.Keys), func(i int) bool {
			return KeyCompare(key, page.Keys[i]) < 0
		})

		path = append(path, pathEntry{pageID: currentID, index: i})
		currentID = page.Children[i]
	}
}

// Search returns the value for a key, whether it was found, and any error.
func (t *Tree) Search(key []byte) ([]byte, bool, error) {
	leaf, _, err := t.findLeaf(key)

	if err != nil {
		return nil, false, err
	}

	i := sort.Search(len(leaf.Keys), func(i int) bool {
		return KeyCompare(leaf.Keys[i], key) >= 0
	})

	if i < len(leaf.Keys) && KeyCompare(leaf.Keys[i], key) == 0 {
		return leaf.Values[i], true, nil
	}

	return nil, false, nil
}

func (t *Tree) Contains(key []byte) (bool, error) {
	_, found, err := t.Search(key)
	return found, err
}

// Insert adds or updates a key/value pair.
func (t *Tree) Insert(key, value []byte) error {
	leaf, path, err := t.findLeaf(key)
	if err != nil {
		return err
	}
	if err := t.insertIntoLeaf(leaf, key, value); err != nil {
		return err
	}

	if len(leaf.Keys) >= maxChildrenPerInternalNode {
		return t.splitLeaf(leaf, path)
	}
	return nil
}

func (t *Tree) insertIntoLeaf(leaf *Page, key, value []byte) error {
	i := sort.Search(len(leaf.Keys), func(i int) bool {
		return KeyCompare(leaf.Keys[i], key) >= 0
	})

	if i < len(leaf.Keys) && KeyCompare(leaf.Keys[i], key) == 0 {
		leaf.Values[i] = value
		leaf.NumKeys = uint16(len(leaf.Keys))
		return t.savePage(leaf)
	}

	leaf.Keys = append(leaf.Keys, nil)
	leaf.Values = append(leaf.Values, nil)
	copy(leaf.Keys[i+1:], leaf.Keys[i:])
	copy(leaf.Values[i+1:], leaf.Values[i:])
	leaf.Keys[i] = key
	leaf.Values[i] = value
	leaf.NumKeys = uint16(len(leaf.Keys))
	return t.savePage(leaf)
}

func (t *Tree) splitLeaf(leaf *Page, path []pathEntry) error {
	mid := len(leaf.Keys) / 2

	rightID := t.pager.NextID()
	right := &Page{
		ID:       rightID,
		IsLeaf:   true,
		NumKeys:  uint16(len(leaf.Keys) - mid),
		Keys:     append([][]byte{}, leaf.Keys[mid:]...),
		Values:   append([][]byte{}, leaf.Values[mid:]...),
		NextLeaf: leaf.NextLeaf,
	}
	leaf.Keys = leaf.Keys[:mid]
	leaf.Values = leaf.Values[:mid]
	leaf.NextLeaf = rightID
	leaf.NumKeys = uint16(len(leaf.Keys))

	if err := t.savePage(leaf); err != nil {
		return err
	}
	if err := t.savePage(right); err != nil {
		return err
	}

	return t.insertIntoParent(leaf.ID, right.Keys[0], rightID, path)
}

func (t *Tree) insertIntoParent(leftID PageID, key []byte, rightID PageID, path []pathEntry) error {
	if len(path) == 0 {
		newRootID := t.pager.NextID()
		newRoot := &Page{
			ID:       newRootID,
			IsLeaf:   false,
			NumKeys:  1,
			Keys:     [][]byte{key},
			Children: []PageID{leftID, rightID},
		}
		if err := t.savePage(newRoot); err != nil {
			return err
		}
		t.rootID = newRootID
		return nil
	}

	parentEntry := path[len(path)-1]
	parent, err := t.loadPage(parentEntry.pageID)
	if err != nil {
		return err
	}

	i := parentEntry.index

	parent.Keys = append(parent.Keys, nil)
	copy(parent.Keys[i+1:], parent.Keys[i:])
	parent.Keys[i] = key

	parent.Children = append(parent.Children, 0)
	copy(parent.Children[i+2:], parent.Children[i+1:])
	parent.Children[i+1] = rightID
	parent.NumKeys = uint16(len(parent.Keys))

	if err := t.savePage(parent); err != nil {
		return err
	}

	if len(parent.Keys) >= maxChildrenPerInternalNode {
		return t.splitInternal(parent, path[:len(path)-1])
	}
	return nil
}

func (t *Tree) splitInternal(n *Page, path []pathEntry) error {
	mid := len(n.Keys) / 2
	upKey := n.Keys[mid]

	rightID := t.pager.NextID()
	right := &Page{
		ID:       rightID,
		IsLeaf:   false,
		Keys:     append([][]byte{}, n.Keys[mid+1:]...),
		Children: append([]PageID{}, n.Children[mid+1:]...),
		NumKeys:  uint16(len(n.Keys) - mid - 1),
	}

	n.Keys = n.Keys[:mid]
	n.Children = n.Children[:mid+1]
	n.NumKeys = uint16(len(n.Keys))

	if err := t.savePage(n); err != nil {
		return err
	}
	if err := t.savePage(right); err != nil {
		return err
	}

	return t.insertIntoParent(n.ID, upKey, rightID, path)
}

func (t *Tree) All() ([][]byte, error) {
	leaf, _, err := t.findLeaf([]byte{0})
	if err != nil {
		return nil, err
	}

	var result [][]byte
	for leaf != nil {
		for _, val := range leaf.Values {
			result = append(result, val)
		}
		if leaf.NextLeaf == 0 {
			break
		}
		leaf, err = t.loadPage(leaf.NextLeaf)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (t *Tree) KeyRange() ([]byte, []byte, bool) {
	first, ok := t.FirstKey()

	if !ok {
		return nil, nil, false
	}

	last, ok := t.LastKey()

	if !ok {
		return nil, nil, false
	}

	return first, last, true
}

func (t *Tree) FirstKey() ([]byte, bool) {
	leaf, _, err := t.findLeaf([]byte{0})

	if err != nil {
		return nil, false
	}

	if len(leaf.Keys) < 1 {
		return nil, false
	}

	return leaf.Keys[0], true
}

func (t *Tree) LastKey() ([]byte, bool) {
	leaf, _, err := t.findLeaf([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	if err != nil || len(leaf.Keys) == 0 {
		return nil, false
	}

	for leaf.NextLeaf != 0 {
		next, err := t.loadPage(leaf.NextLeaf)
		if err != nil || len(next.Keys) == 0 {
			break
		}
		leaf = next
	}

	return leaf.Keys[len(leaf.Keys)-1], true
}
