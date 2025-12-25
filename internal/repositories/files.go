package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type File struct {
	BaseModel

	digest      string
	contentType string
	data        []byte
	size        int64
}

func NewFile(digest string, contentType string, data []byte) *File {
	return &File{
		BaseModel:   NewBaseModel(),
		digest:      digest,
		contentType: contentType,
		data:        data,
		size:        int64(len(data)),
	}
}

func NewFileFromDB(digest string, contentType string, data []byte, size int64, base BaseModel) *File {
	return &File{
		BaseModel:   base,
		digest:      digest,
		contentType: contentType,
		data:        data,
		size:        size,
	}
}

func (f *File) GetDigest() string {
	return f.digest
}

func (f *File) GetContentType() string {
	return f.contentType
}

func (f *File) GetData() []byte {
	return f.data
}

func (f *File) GetSize() int64 {
	return f.size
}

type FileFilter struct {
	Id     *uuid.UUID
	Digest *string
}

func NewFileFilter() *FileFilter {
	return &FileFilter{}
}

func (f *FileFilter) clone() *FileFilter {
	cloned := *f
	return &cloned
}

func (f *FileFilter) ById(id uuid.UUID) *FileFilter {
	cloned := f.clone()
	cloned.Id = &id
	return cloned
}

func (f *FileFilter) HasId() bool {
	return f.Id != nil
}

func (f *FileFilter) GetId() uuid.UUID {
	return pointer.DerefOrZero(f.Id)
}

func (f *FileFilter) ByDigest(digest string) *FileFilter {
	cloned := f.clone()
	cloned.Digest = &digest
	return cloned
}

func (f *FileFilter) HasDigest() bool {
	return f.Digest != nil
}

func (f *FileFilter) GetDigest() string {
	return pointer.DerefOrZero(f.Digest)
}

type FileRepository interface {
	Single(ctx context.Context, filter *FileFilter) (*File, error)
	First(ctx context.Context, filter *FileFilter) (*File, error)
	List(ctx context.Context, filter *FileFilter) ([]*File, int, error)
	Insert(file *File)
	Delete(file *File)
}
