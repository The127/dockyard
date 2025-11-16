package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/the127/dockyard/internal/utils/pointer"
)

type File struct {
	BaseModel

	Digest      string
	ContentType string
	Data        []byte
	Size        int64
}

func NewFile(digest string, contentType string, data []byte) *File {
	return &File{
		BaseModel:   NewBaseModel(),
		Digest:      digest,
		ContentType: contentType,
		Data:        data,
		Size:        int64(len(data)),
	}
}

func (f *File) GetDigest() string {
	return f.Digest
}

func (f *File) GetContentType() string {
	return f.ContentType
}

func (f *File) GetData() []byte {
	return f.Data
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
	Insert(ctx context.Context, file *File) error
	Update(ctx context.Context, file *File) error
	Delete(ctx context.Context, id uuid.UUID) error
}
