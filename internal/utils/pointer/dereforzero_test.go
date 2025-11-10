package pointer

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DerefOrZeroTestSuite struct {
	suite.Suite
}

func TestDerefOrZeroTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DerefOrZeroTestSuite))
}

func (s *DerefOrZeroTestSuite) TestNonNil() {
	// arrange
	value := "value"
	ptr := &value

	// act
	actual := DerefOrZero(ptr)

	// assert
	s.Equal(value, actual)
}

func (s *DerefOrZeroTestSuite) TestNilInt() {
	// arrange
	var expected int
	var ptr *int = nil

	// act
	actual := DerefOrZero(ptr)

	// assert
	s.Equal(expected, actual)
}

func (s *DerefOrZeroTestSuite) TestNilString() {
	// arrange
	var expected string
	var ptr *string = nil

	// act
	actual := DerefOrZero(ptr)

	// assert
	s.Equal(expected, actual)
}

func (s *DerefOrZeroTestSuite) TestNilStruct() {
	// arrange
	var expected testStruct
	var ptr *testStruct = nil

	// act
	actual := DerefOrZero(ptr)

	// assert
	s.Equal(expected, actual)
}
