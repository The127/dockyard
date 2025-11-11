package jsontypes

import "github.com/google/uuid"

type BlobUploadMode string

const (
	BlobUploadModeMonolithic BlobUploadMode = "monolithic"
	BlobUploadModeChunked    BlobUploadMode = "chunked"
)

type UploadSession struct {
	Id             uuid.UUID      `json:"id"`
	UploadMode     BlobUploadMode `json:"uploadMode"`
	TenantSlug     string         `json:"tenantSlug"`
	ProjectSlug    string         `json:"projectSlug"`
	RepositorySlug string         `json:"repositorySlug"`
	RepositoryId   uuid.UUID      `json:"repositoryId"`
	RangeEnd       int64          `json:"rangeEnd"`
	DigestState    []byte         `json:"digestState"`
}
