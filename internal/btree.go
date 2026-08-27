package internal

import (
	"encoding/binary"
	"errors"
	"fmt"
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

	page.ID = id

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

func (t *Tree) InsertNext(value []byte) (RowID, error) {
	rowID, err := t.NextRowID()

	if err != nil {
		return 0, err
	}

	err = t.InsertAt(rowID, value)

	return rowID, err
}

func (t *Tree) InsertAt(rowID RowID, value []byte) error {
	key := EncodeKey(rowID)

	return t.Insert(key, value)
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

	if len(path) == 0 {
		// Splitting the root leaf: keep the root page ID stable by rewriting
		// it as an internal node and moving the left half to a fresh page, so
		// the schema's rootpage pointer never goes stale.
		newLeftID := t.pager.NextID()
		left := &Page{
			ID:       newLeftID,
			IsLeaf:   true,
			NumKeys:  uint16(mid),
			Keys:     append([][]byte{}, leaf.Keys[:mid]...),
			Values:   append([][]byte{}, leaf.Values[:mid]...),
			NextLeaf: rightID,
		}

		if err := t.savePage(left); err != nil {
			return err
		}

		if err := t.savePage(right); err != nil {
			return err
		}

		root := &Page{
			ID:       t.rootID,
			IsLeaf:   false,
			NumKeys:  1,
			Keys:     [][]byte{right.Keys[0]},
			Children: []PageID{newLeftID, rightID},
		}

		return t.savePage(root)
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
		// The old root (leftID) split and its left half was already saved.
		// Keep the root page ID stable: move the left half to a fresh page and
		// rewrite the old root page as the new internal root, so the schema's
		// rootpage pointer never goes stale.
		newLeftID := t.pager.NextID()
		left, err := t.loadPage(leftID)
		if err != nil {
			return err
		}
		left.ID = newLeftID
		if err := t.savePage(left); err != nil {
			return err
		}

		newRoot := &Page{
			ID:       t.rootID,
			IsLeaf:   false,
			NumKeys:  1,
			Keys:     [][]byte{key},
			Children: []PageID{newLeftID, rightID},
		}
		return t.savePage(newRoot)
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

func (t *Tree) All() ([][]byte, [][]byte, error) {
	leaf, _, err := t.findLeaf([]byte{0})
	if err != nil {
		return nil, nil, err
	}

	var keys, result [][]byte
	for leaf != nil {
		keys = append(keys, leaf.Keys...)
		result = append(result, leaf.Values...)
		if leaf.NextLeaf == 0 {
			break
		}
		leaf, err = t.loadPage(leaf.NextLeaf)
		if err != nil {
			return nil, nil, err
		}
	}
	return keys, result, nil
}

// Delete removes a key and its value from the tree.
func (t *Tree) Delete(key []byte) error {
	leaf, _, err := t.findLeaf(key)
	if err != nil {
		return err
	}

	i := sort.Search(len(leaf.Keys), func(i int) bool {
		return KeyCompare(leaf.Keys[i], key) >= 0
	})

	if i >= len(leaf.Keys) || KeyCompare(leaf.Keys[i], key) != 0 {
		return fmt.Errorf("key %v not found", key)
	}

	leaf.Keys = append(leaf.Keys[:i], leaf.Keys[i+1:]...)
	leaf.Values = append(leaf.Values[:i], leaf.Values[i+1:]...)
	leaf.NumKeys = uint16(len(leaf.Keys))

	return t.savePage(leaf)
}

func (t *Tree) KeyRange() ([]byte, []byte, error) {
	first, err := t.FirstKey()

	if err != nil {
		return nil, nil, err
	}

	last, err := t.LastKey()

	if err != nil {
		return nil, nil, err
	}

	return first, last, nil
}

func (t *Tree) FirstKey() ([]byte, error) {
	leaf, _, err := t.findLeaf([]byte{0})

	if err != nil {
		return nil, err
	}

	if len(leaf.Keys) < 1 {
		return EncodeKey(0), nil
	}

	return leaf.Keys[0], nil
}

func (t *Tree) LastKey() ([]byte, error) {
	leaf, _, err := t.findLeaf([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	if err != nil {
		return nil, err
	}

	if len(leaf.Keys) == 0 {
		return EncodeKey(0), nil
	}

	for leaf.NextLeaf != 0 {
		next, err := t.loadPage(leaf.NextLeaf)
		if err != nil || len(next.Keys) == 0 {
			break
		}
		leaf = next
	}

	return leaf.Keys[len(leaf.Keys)-1], nil
}

func (t *Tree) NextRowID() (RowID, error) {
	buf, err := t.LastKey()

	if err != nil {
		return 0, err
	}

	lastKey := DecodeKey(buf)
	next := lastKey + 1

	return next, nil
}

func EncodeKey(id RowID) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(id))
	return buf
}

func DecodeKey(buf []byte) RowID {
	if len(buf) < 8 {
		panic(fmt.Errorf("cannot decode key from buffer of length %v", len(buf)))
	}

	key := binary.BigEndian.Uint64(buf)
	return RowID(key)
}
