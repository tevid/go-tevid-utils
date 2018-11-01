package db_scan

import (
	. "github.com/tevid/gohamcrest"
	"testing"
)

func TestDBScanExtraSigleData(t *testing.T) {

	type Person struct {
		Name string `json:"pname" pg:"pname"`
		Age  int    `json:"page" pg:"page"`
	}

	var p Person
	name := "tencent"
	age := 20

	var mp = map[string]interface{}{
		"pname": name,
		"page":  age,
	}

	err := singleResult(mp, &p)

	Assert(t, err, NilVal())

	Assert(t, name, Equal(p.Name))

	Assert(t, age, Equal(p.Age))
}

func TestDBScanExtraSigleDataUsingByteString(t *testing.T) {
	type Person struct {
		Name string `pg:"name"`
		Age  int    `pg:"ag"`
	}
	var p Person
	name := []byte{'t', 'e', 'n', 'c', 'e', 'n', 't'}
	age := 20
	var mp = map[string]interface{}{
		"name": name,
		"ag":   age,
	}
	err := singleResult(mp, &p)
	Assert(t, err, NilVal())

	Assert(t, string(name), Equal(p.Name))

	Assert(t, age, Equal(p.Age))
}

func TestDBScanExtraMultiData(t *testing.T) {
	type Stu struct {
		Age int `pg:"age"`
	}
	var students []Stu
	testCases := []int{1, 2, 3, 4, 5, 6, 9, 0, 7, 8}
	var data []map[string]interface{}
	for _, v := range testCases {
		data = append(data, map[string]interface{}{"age": v})
	}
	err := multiResults(data, &students)

	if err != nil {
		t.Fail()
	}

	Assert(t, err, NilVal())

	Assert(t, len(testCases), Equal(len(students)))

	for idx, p := range students {
		if testCases[idx] != p.Age {
			t.Fail()
		}
	}
}
