package rpctest

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type callTestClient struct{ calls int }

func (c *callTestClient) Fixed(n int) (int, error) { c.calls++; return n, nil }
func (c *callTestClient) Variadic(n int, values ...int) (int, error) {
	c.calls++
	for _, v := range values {
		n += v
	}
	return n, nil
}
func (c *callTestClient) Null(v interface{}) (interface{}, error) { c.calls++; return v, nil }
func (c *callTestClient) BadResult()                              { c.calls++ }

func TestReflectCallValidation(t *testing.T) {
	for _, tc := range []struct {
		name, method string
		params       []interface{}
		want         int
		invalid      bool
	}{
		{"fixed", "Fixed", []interface{}{3}, 3, false},
		{"too few", "Fixed", nil, 0, true},
		{"too many", "Fixed", []interface{}{1, 2}, 0, true},
		{"variadic empty", "Variadic", []interface{}{1}, 1, false},
		{"variadic one", "Variadic", []interface{}{1, 2}, 3, false},
		{"variadic multiple", "Variadic", []interface{}{1, 2, 3}, 6, false},
		{"variadic missing fixed", "Variadic", nil, 0, true},
		{"bad conversion", "Fixed", []interface{}{"not a number"}, 0, true},
		{"missing method", "Missing", nil, 0, true},
		{"bad result count", "BadResult", nil, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &callTestClient{}
			got, rpcErr, err := reflectCall(c, tc.method, tc.params)
			if tc.invalid {
				if err == nil || c.calls != 0 {
					t.Fatalf("error=%v calls=%d; want error before invocation", err, c.calls)
				}
				return
			}
			if err != nil || rpcErr != nil || got != tc.want || c.calls != 1 {
				t.Fatalf("result=%v rpcError=%v error=%v calls=%d", got, rpcErr, err, c.calls)
			}
		})
	}
}

func TestReflectCallNilValues(t *testing.T) {
	for _, c := range []interface{}{nil, (*callTestClient)(nil)} {
		if _, _, err := reflectCall(c, "Fixed", []interface{}{1}); err == nil {
			t.Fatal("nil client accepted")
		}
	}
	got, rpcErr, err := reflectCall(&callTestClient{}, "Null", []interface{}{nil})
	if got != nil || rpcErr != nil || err != nil {
		t.Fatalf("nil interface result=%v rpcError=%v error=%v", got, rpcErr, err)
	}
}

func TestMockExamplesSizeLimit(t *testing.T) {
	const limit = 16 * 1024 * 1024
	for _, size := range []int{limit - 1, limit, limit + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Flush to force a chunked response without a Content-Length header.
				w.(http.Flusher).Flush()
				_, _ = io.WriteString(w, "{}"+strings.Repeat(" ", size-2))
			}))
			defer server.Close()
			got, err := getMockExamples(server.URL)
			if size > limit {
				if err == nil || !strings.Contains(err.Error(), "16777216") {
					t.Fatalf("oversized response error=%v", err)
				}
				return
			}
			if err != nil || got == nil {
				t.Fatalf("size=%d result=%v error=%v", size, got, err)
			}
		})
	}
}
