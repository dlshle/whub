package controllers

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"lastModified"`
	IsDir   bool      `json:"isDir"`
}

func NewFileInfo(info os.FileInfo) FileInfo {
	return FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}
}

type IFileController interface {
	Info(path string) (FileInfo, error)
	Read(path string, sec int) ([]byte, error)
	List(path string) ([]FileInfo, error)
	GetFile(path string) ([]byte, error)
}

type FileController struct {
	rootDir     string
	sectionSize int64
}

func NewFileController(rootDir string, sectionSize int64) (IFileController, error) {
	rootDir, err := getRootDirOrCurrentDir(rootDir)
	if err != nil {
		return nil, err
	}
	// if root dir does not exist, try to mkdir
	if _, err = os.Lstat(rootDir); os.IsNotExist(err) {
		err = os.Mkdir(fmt.Sprintf("%s/", rootDir), os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	_, err = os.Open(rootDir)
	if err != nil {
		return nil, err
	}
	return &FileController{
		rootDir:     rootDir,
		sectionSize: sectionSize,
	}, nil
}

func getRootDirOrCurrentDir(rootDir string) (rootPath string, err error) {
	rootPath = rootDir
	if rootDir == "" || rootDir == "/" {
		rootPath, err = GetCurrentPath()
	}
	return
}

func (c *FileController) open(path string) (*os.File, error) {
	return os.Open(fmt.Sprintf("%s%s", c.rootDir, path))
}

func (c *FileController) Info(path string) (info FileInfo, err error) {
	file, err := c.open(path)
	if err != nil {
		return
	}
	stat, err := file.Stat()
	if err != nil {
		return
	}
	return NewFileInfo(stat), nil
}

func (c *FileController) read(file *os.File, from int64, numBytes int64) ([]byte, error) {
	buffer := make([]byte, numBytes, numBytes)
	_, err := file.ReadAt(buffer, from)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func (c *FileController) readSection(file *os.File, sec int) ([]byte, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	begin := (int64)(sec) * c.sectionSize
	if begin > stat.Size() {
		return []byte{}, nil
	}
	return c.read(file, begin, c.sectionSize)
}

func (c *FileController) Read(path string, sec int) ([]byte, error) {
	file, err := c.open(path)
	if err != nil {
		return nil, err
	}
	return c.readSection(file, sec)
}

func (c *FileController) List(path string) ([]FileInfo, error) {
	file, err := c.open(path)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New(fmt.Sprintf("path %s is not a directory", path))
	}
	stats, err := file.Readdir(0)
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return []FileInfo{}, nil
	}
	infos := make([]FileInfo, len(stats), len(stats))
	for i, s := range stats {
		infos[i] = NewFileInfo(s)
	}
	return infos, nil
}

func (c *FileController) GetFile(path string) ([]byte, error) {
	file, err := c.open(path)
	if err != nil {
		return nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() > c.sectionSize*5 {
		return nil, errors.New("file is too large to request in one time")
	}
	return c.read(file, 0, stat.Size())
}

func GetCurrentPath() (string, error) {
	return os.Getwd()
}
