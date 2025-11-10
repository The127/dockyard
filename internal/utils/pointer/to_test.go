package pointer

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ToTestSuite struct {
	suite.Suite
}

func TestToTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ToTestSuite))
}

func (s *ToTestSuite) TestInt() {
	// arrange
	var v = 0
	var expected = &v

	// act
	actual := To(v)

	// assert
	s.Equal(expected, actual)
}

func (s *ToTestSuite) TestString() {
	// arrange
	var v = "string"
	var expected = &v

	// act
	actual := To(v)

	// assert
	s.Equal(expected, actual)
}

func (s *ToTestSuite) TestStruct() {
	// arrange
	var v = testStruct{field: "field"}
	var expected = &v

	// act
	actual := To(v)

	// assert
	s.Equal(expected, actual)
}

func (s *ToTestSuite) TestStructPointer() {
	// arrange
	var v = &testStruct{field: "field"}
	var expected = &v

	// act
	actual := To(v)

	// assert
	s.Equal(expected, actual)
}
