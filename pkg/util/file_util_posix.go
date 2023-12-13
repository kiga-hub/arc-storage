//go:build windows
// +build windows

package util

import (
	"os"
)

// GetFileUIDGid -
func GetFileUIDGid(fi os.FileInfo) (uid, gid uint32) {
	return 0, 0
}
