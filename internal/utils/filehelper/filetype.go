package filehelper

import (
	"slices"
	"strings"

	"github.com/morya/spgw/internal/global/consts"
)

// GetFileTypeBySuffix 根据文件后缀判断文件类型
func GetFileTypeBySuffix(suffix string) string {
	// 标准化后缀，确保以点开头并转为小写
	if !strings.HasPrefix(suffix, ".") {
		suffix = "." + suffix
	}
	suffix = strings.ToLower(suffix)

	// 图片类型
	imageTypes := []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico",
	}

	// 视频类型
	videoTypes := []string{
		".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".webm", ".m4v",
	}

	// 音频类型
	audioTypes := []string{
		".mp3", ".wav", ".wma", ".ogg", ".aac", ".flac", ".m4a",
	}

	// 文档类型
	documentTypes := []string{
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".txt", ".rtf", ".md", ".csv",
	}

	// 压缩文件类型
	archiveTypes := []string{
		".zip", ".rar", ".7z", ".tar", ".gz", ".bz2",
	}

	// 检查文件类型
	if slices.Contains(imageTypes, suffix) {
		return consts.UploadFileTypeImage
	}

	if slices.Contains(videoTypes, suffix) {
		return consts.UploadFileTypeVideo
	}

	if slices.Contains(audioTypes, suffix) {
		return consts.UploadFileTypeAudio
	}

	if slices.Contains(documentTypes, suffix) {
		return consts.UploadFileTypeDocument
	}

	if slices.Contains(archiveTypes, suffix) {
		return "archive"
	}

	// 默认返回其他类型
	return consts.UploadFileTypeOther
}
