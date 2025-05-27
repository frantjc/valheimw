package command

import (
	"fmt"
	"strconv"

	"github.com/spf13/pflag"
	"golang.org/x/exp/constraints"
)

type Bool[T any] struct {
	Value *T
	IfSet T
}

var (
	_ pflag.Value = new(Bool[any])
)

// Set implements pflag.Value.
func (b *Bool[T]) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if v {
		*b.Value = b.IfSet
	}
	return err
}

// String implements pflag.Value.
func (b *Bool[T]) String() string {
	return fmt.Sprint(b.Value)
}

// Type implements pflag.Value.
func (b *Bool[T]) Type() string {
	return "bool"
}

// Type implements pflag.boolFlag.
func (b *Bool[T]) IsBoolFlag() bool {
	return true
}

type Count[T constraints.Integer] struct {
	Value     *T
	Increment T
}

var (
	_ pflag.Value = new(Count[int])
)

// Set implements pflag.Value.
func (c *Count[T]) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 0)
	*c.Value += (T(v) * c.Increment)
	return err
}

// String implements pflag.Value.
func (c *Count[T]) String() string {
	return strconv.Itoa(int(*c.Value))
}

// Type implements pflag.Value.
func (c *Count[T]) Type() string {
	return "count"
}
