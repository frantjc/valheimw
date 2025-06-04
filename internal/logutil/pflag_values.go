package logutil

import (
	"fmt"
	"strconv"

	"github.com/spf13/pflag"
	"golang.org/x/exp/constraints"
)

type genericBool[T any] struct {
	Value *T
	IfSet T
}

var (
	_ pflag.Value = new(genericBool[any])
)

// Set implements pflag.Value.
func (b *genericBool[T]) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if v {
		*b.Value = b.IfSet
	}
	return err
}

// String implements pflag.Value.
func (b *genericBool[T]) String() string {
	return fmt.Sprint(b.Value)
}

// Type implements pflag.Value.
func (b *genericBool[T]) Type() string {
	return "bool"
}

// Type implements pflag.boolFlag.
func (b *genericBool[T]) IsBoolFlag() bool {
	return true
}

type incrementalCount[T constraints.Integer] struct {
	Value     *T
	Increment T
}

var (
	_ pflag.Value = new(incrementalCount[int])
)

// Set implements pflag.Value.
func (c *incrementalCount[T]) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 0)
	*c.Value += (T(v) * c.Increment)
	return err
}

// String implements pflag.Value.
func (c *incrementalCount[T]) String() string {
	return strconv.Itoa(int(*c.Value))
}

// Type implements pflag.Value.
func (c *incrementalCount[T]) Type() string {
	return "count"
}
