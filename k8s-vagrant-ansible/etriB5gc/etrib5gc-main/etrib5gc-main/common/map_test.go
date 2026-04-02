package common

import (
	"fmt"
	"testing"
)

type MapItem struct {
	id string
	v  int
}

func itemIndex(i *MapItem) string {
	return i.id
}

func (i MapItem) MapIndex() string {
	return i.id
}

func TestSimpleItemMap(t *testing.T) {
	fmt.Println("Test SimpleItemMap")
	items := NewSimpleMap[string, MapItem](itemIndex)
	i := &MapItem{
		id: "1",
		v:  1,
	}
	items.Add(i)
	j := &MapItem{
		id: "2",
		v:  2,
	}
	items.Add(j)
	items.Add(&MapItem{
		id: "3",
		v:  3,
	})

	if item := items.Find("1"); item == nil {
		t.Fatalf("item not found")
	} else {
		if item.v != 1 {
			t.Fatalf("wrong item found")
		}
		i.v = 3
		if item.v != 3 {
			t.Fatalf("wrong item pointer")
		}
	}
	items.Remove("1")

	if item := items.Find("1"); item != nil {
		t.Fatalf("item not delete")
	}

	all := items.ListAll()
	j.v = 10
	for _, i := range all {
		fmt.Printf("%s-%d\n", i.id, i.v)
	}
}

type MultiMapItem struct {
	id1 string
	id2 string
	id3 string
	v   int
}
type ItemMultiMap struct {
	MultiMap[MultiMapItem]
}

func newItemMultiMap() ItemMultiMap {
	return ItemMultiMap{
		MultiMap: NewMultiMap[MultiMapItem]([]uint8{1, 2}, getItemIndex),
	}
}

func getItemIndex(i *MultiMapItem, t uint8) string {
	switch t {
	case 1:
		return i.id1
	case 2:
		return i.id2
	}
	return ""
}

func TestMultiMap(t *testing.T) {
	fmt.Println("Test MultiItemMap")
	items := newItemMultiMap()
	i1 := &MultiMapItem{
		id1: "1-1",
		id2: "2-1",
		v:   1,
	}
	i2 := &MultiMapItem{
		id1: "1-2",
		id2: "2-2",
		v:   2,
	}
	i3 := &MultiMapItem{
		id1: "1-3",
		//id2: "2-3",
		v: 3,
	}
	items.Update(i1)
	items.Update(i2)
	items.Update(i3)

	if i := items.Find("2-3", 2); i != nil {
		t.Fatal("found non-existed item")
	}
	i3.id2 = "2-3"
	items.Update(i3)
	if i := items.Find("2-3", 2); i == nil {
		t.Fatal("existed item not found")
	}
	items.Remove(i2)
	i1.v = 10
	items.Update(i1)
	all := items.ListAll(1)
	if len(all) != 2 {
		t.Fatal("Failed to delete item")
	}
	i3.v = 100
	for _, i := range all {
		fmt.Printf("%v\n", *i)
	}
}
