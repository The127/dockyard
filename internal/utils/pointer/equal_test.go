package pointer

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EqualTestSuite struct {
	suite.Suite
}

func TestEqualTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(EqualTestSuite))
}

func (s *EqualTestSuite) TestBothNil() {
	var a, b *string

	s.True(Equal(a, b))
}

func (s *EqualTestSuite) TestFirstNil() {
	var a *string
	b := To("x")

	s.False(Equal(a, b))
}

func (s *EqualTestSuite) TestSecondNil() {
	a := To("x")
	var b *string

	s.False(Equal(a, b))
}

func (s *EqualTestSuite) TestSameValue() {
	a := To("hello")
	b := To("hello")

	s.True(Equal(a, b))
}

func (s *EqualTestSuite) TestDifferentValue() {
	a := To("hello")
	b := To("world")

	s.False(Equal(a, b))
}

func (s *EqualTestSuite) TestSamePointer() {
	a := To("hello")

	s.True(Equal(a, a))
}

func (s *EqualTestSuite) TestInt() {
	a := To(42)
	b := To(42)
	c := To(99)

	s.True(Equal(a, b))
	s.False(Equal(a, c))
}
