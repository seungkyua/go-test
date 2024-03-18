package embed_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/seungkyua/go-test/interface/embed"
)

type MathResolverStub struct{}

func (mr MathResolverStub) Resolve(expr string) (float64, error) {
	switch expr {
	case "2 + 4 * 10":
		return 42, nil
	case "( 2 + 4 ) * 10":
		return 60, nil
	case "( 2 + 4 * 10":
		return 0, fmt.Errorf("invalid expression: %s", expr)
	}
	return 0, nil
}

func TestCalculatorProcess(t *testing.T) {
	c := embed.Calculator{Resolver: MathResolverStub{}}
	in := strings.NewReader(`2 + 4 * 10
( 2 + 4 ) * 10
( 2 + 4 * 10`)

	data := []float64{42, 60, 0}
	expectedErr := errors.New("invalid expression: ( 2 + 4 * 10")
	for _, d := range data {
		result, err := c.Process(in)
		if err != nil {
			if err.Error() != expectedErr.Error() {
				t.Errorf("want (%v) got (%v)", expectedErr, err)
			}
		}
		if result != d {
			t.Errorf("Expected result %f, got %f", d, result)
		}
	}
}
