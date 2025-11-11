package inmemory

import (
	"fmt"

	"github.com/google/uuid"
)

type UUIDValueIndexer struct {
	Getter func(obj interface{}) uuid.UUID
}

func (u *UUIDValueIndexer) FromObject(obj interface{}) (bool, []byte, error) {
	val := u.Getter(obj)
	if val == uuid.Nil {
		return false, nil, nil
	}

	buf, err := val.MarshalBinary()
	return true, buf, err
}

func (u *UUIDValueIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("UUIDValueIndexer takes exactly one argument")
	}

	id, ok := args[0].(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("argument is not uuid.UUID")
	}

	return []byte(id.String()), nil
}
