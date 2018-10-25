package ordered_map

import (
	"testing"

	. "github.com/tevid/gohamcrest"
)

func TestOrderedMap(t *testing.T) {

	testorderedmap := New("t", "e", "v", "i", "d", "d", "e", "v", "l", "o", "p")

	Assert(t, testorderedmap.Exist("v"), Equal(true))
	Assert(t, testorderedmap.Exist("t"), Equal(true))
	Assert(t, testorderedmap.Exist("x"), Not(Equal(true)))

	data, err := testorderedmap.MarshalJSON()

	Assert(t, err, Equal(nil))
	Assert(t, `["t","e","v","i","d","l","o","p"]`, Equal(string(data)))
}
