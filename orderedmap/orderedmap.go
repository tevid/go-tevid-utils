package orderedmap

import (
	"encoding/json"
)

type OrderedMap struct {
	datamap   map[interface{}]bool
	dataslice []interface{}
}

func New(vs ...interface{}) (t OrderedMap) {
	t.ensureDataStruct()
	for _, v := range vs {
		t.Add(v)
	}
	return
}

func (t *OrderedMap) ensureDataStruct() {
	if t.datamap == nil {
		t.datamap = make(map[interface{}]bool)
	}
	if t.dataslice == nil {
		t.dataslice = []interface{}{}
	}
}

func (t *OrderedMap) Add(v interface{}) bool {
	t.ensureDataStruct()
	_, ok := t.datamap[v]
	if ok {
		return false
	} else {
		t.datamap[v] = true
		t.dataslice = append(t.dataslice, v)
		return true
	}
}

func (t *OrderedMap) Remove(v interface{}) bool {
	t.ensureDataStruct()
	_, ok := t.datamap[v]
	if !ok {
		return false
	}
	delete(t.datamap, v)
	for i, vinslice := range t.dataslice {
		if v == vinslice {
			t.dataslice = append(t.dataslice[:i], t.dataslice[i+1:]...)
			break
		}
	}
	return true
}

func (t OrderedMap) IsEmpty() bool {
	t.ensureDataStruct()
	return len(t.dataslice) < 1
}

func (t OrderedMap) Exist(v interface{}) bool {
	t.ensureDataStruct()
	_, exist := t.datamap[v]
	return exist
}

func (t OrderedMap) Map() map[interface{}]bool {
	t.ensureDataStruct()
	return t.datamap
}

func (t OrderedMap) Slice() []interface{} {
	t.ensureDataStruct()
	return t.dataslice
}

func (t *OrderedMap) Len() int {
	if t == nil {
		return 0
	}
	t.ensureDataStruct()
	return len(t.dataslice)
}

func (t OrderedMap) MarshalJSON() ([]byte, error) {
	t.ensureDataStruct()
	return json.Marshal(t.Slice())
}

func (t *OrderedMap) UnmarshalJSON(data []byte) (err error) {
	var (
		dataslice []interface{}
	)

	t.ensureDataStruct()

	if err = json.Unmarshal(data, &dataslice); err != nil {
		return
	}

	t.datamap = map[interface{}]bool{}
	t.dataslice = []interface{}{}

	for _, v := range dataslice {
		t.Add(v)
	}

	return
}
