package models

import (
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListRequest 文件附件列表请求参数
type ListRequest struct {
	BasePaging
	Validator
	Name  string `json:"name" form:"name"`   // 文件名
	Ftype string `json:"ftype" form:"ftype"` // 文件类型
}

// Validate 验证请求参数
func (r *ListRequest) Validate(c *gin.Context) error {
	return r.Validator.Check(c, r)
}

// GetQuery 获取查询条件
func (r *ListRequest) Handle() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if r.Name != "" {
			db = db.Where("name LIKE ?", "%"+r.Name+"%")
		}
		if r.Ftype != "" {
			db = db.Where("ftype = ?", r.Ftype)
		}
		return db
	}
}

// UploadRequest 上传文件请求参数
type UploadRequest struct {
	Validator
	File    *multipart.FileHeader `form:"file" validate:"required"`
	IsThumb int                   `form:"isThumb"`
	Width   int                   `form:"width"`
	Height  int                   `form:"height"`
}

// Validate 验证请求参数
func (r *UploadRequest) Validate(c *gin.Context) error {
	return r.Validator.Check(c, r)
}

// AffixDeleteRequest 删除文件请求参数
type AffixDeleteRequest struct {
	Validator
	ID uint `json:"id" form:"id" validate:"required"`
}

// Validate 验证请求参数
func (r *AffixDeleteRequest) Validate(c *gin.Context) error {
	return r.Validator.Check(c, r)
}

// UpdateNameRequest 修改文件名请求参数
type UpdateNameRequest struct {
	Validator
	ID   uint   `json:"id" form:"id" validate:"required"`
	Name string `json:"name" form:"name" validate:"required"`
}

// Validate 验证请求参数
func (r *UpdateNameRequest) Validate(c *gin.Context) error {
	return r.Validator.Check(c, r)
}

// ===== 分片上传请求参数 =====

// ChunkInitRequest 分片上传初始化请求
type ChunkInitRequest struct {
	Validator
	FileMd5     string `json:"fileMd5" validate:"required"`
	FileName    string `json:"fileName" validate:"required"`
	FileSize    int64  `json:"fileSize" validate:"required"`
	ChunkSize   int    `json:"chunkSize" validate:"required"`
	TotalChunks int    `json:"totalChunks" validate:"required"`
}

// Validate 验证请求参数
func (r *ChunkInitRequest) Validate(c *gin.Context) error {
	return r.Validator.Check(c, r)
}

// ChunkUploadRequest 分片上传请求（multipart/form-data）
type ChunkUploadRequest struct {
	Validator
	File        *multipart.FileHeader `form:"file" validate:"required"`
	UploadId    string                `form:"uploadId" validate:"required"`
	ChunkIndex  int                   `form:"chunkIndex" validate:"required"`
	ChunkMd5    string                `form:"chunkMd5"`
	FileMd5     string                `form:"fileMd5"`
	TotalChunks int                   `form:"totalChunks" validate:"required"`
}

// Validate 验证请求参数
func (r *ChunkUploadRequest) Validate(c *gin.Context) error {
	return r.Validator.Check(c, r)
}

// ChunkMergeRequest 合并分片请求
type ChunkMergeRequest struct {
	Validator
	UploadId    string `json:"uploadId" validate:"required"`
	FileMd5     string `json:"fileMd5" validate:"required"`
	FileName    string `json:"fileName" validate:"required"`
	FileSize    int64  `json:"fileSize" validate:"required"`
	TotalChunks int    `json:"totalChunks" validate:"required"`
}

// Validate 验证请求参数
func (r *ChunkMergeRequest) Validate(c *gin.Context) error {
	return r.Validator.Check(c, r)
}

// ChunkCancelRequest 取消上传请求
type ChunkCancelRequest struct {
	Validator
	UploadId string `json:"uploadId" validate:"required"`
}

// Validate 验证请求参数
func (r *ChunkCancelRequest) Validate(c *gin.Context) error {
	return r.Validator.Check(c, r)
}

// ChunkInitResult 分片上传初始化结果
type ChunkInitResult struct {
	UploadId       string
	UploadedChunks []int
	ExistFile      *SysAffix
}
