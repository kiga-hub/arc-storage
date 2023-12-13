package util

import (
	"path/filepath"
	"strings"
)

// FullPath string
type FullPath string

// NewFullPath return full path
func NewFullPath(dir, name string) FullPath {
	return FullPath(dir).Child(name)
}

// DirAndName return dir and name
func (fp FullPath) DirAndName() (string, string) {
	dir, name := filepath.Split(string(fp))
	if dir == "/" {
		return dir, name
	}
	if len(dir) < 1 {
		return "/", ""
	}
	return dir[:len(dir)-1], name
}

// Name  return file name
func (fp FullPath) Name() string {
	_, name := filepath.Split(string(fp))
	return name
}

// Child -
func (fp FullPath) Child(name string) FullPath {
	dir := string(fp)
	if strings.HasSuffix(dir, "/") {
		return FullPath(dir + name)
	}
	return FullPath(dir + "/" + name)
}

// AsInode return HashStringToLong
func (fp FullPath) AsInode() uint64 {
	v, err := HashStringToLong(string(fp))
	if err != nil {
		return uint64(v)
	}
	return uint64(v)
}

// Split split, but skipping the root
func (fp FullPath) Split() []string {
	if fp == "" || fp == "/" {
		return []string{}
	}
	return strings.Split(string(fp)[1:], "/")
}

// Join file operation
func Join(names ...string) string {
	return filepath.ToSlash(filepath.Join(names...))
}

// JoinPath file operation
func JoinPath(names ...string) FullPath {
	return FullPath(Join(names...))
}
