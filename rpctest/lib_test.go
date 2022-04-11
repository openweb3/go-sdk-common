package rpctest

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

type student struct {
	Name     string
	Treasure *[]byte `json:"treasure,omitempty" testomit:"false"`
	Nicon    *string `json:"nicon" testomit:"true"`
}
type teacher struct {
	Name     string
	Students []student
}

func TestJsonMarshalForTest(t *testing.T) {
	table := []struct {
		raw    interface{}
		expect string
	}{
		{
			raw: teacher{Name: "sophia",
				Students: []student{
					{
						Name:     "jack",
						Treasure: nil,
						Nicon:    nil,
					},
				},
			},
			expect: `{"Name":"sophia","Students":[{"Name":"jack","treasure":null}]}`,
		},
	}

	j, e := jsonMarshalForTest(table[0].raw, false)
	if e != nil {
		panic(e)
	}
	fmt.Printf("marshal result: %s", j)
	assert.Equal(t, table[0].expect, string(j))
}
