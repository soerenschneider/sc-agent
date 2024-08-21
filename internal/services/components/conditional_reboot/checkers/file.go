package checkers

import (
	"context"
	"errors"
	"fmt"
	"os"
)

const FileCheckerName = "file"

type FileChecker struct {
	file         string
	wantsAbsence bool
}

func NewFileChecker(file string) (*FileChecker, error) {
	if len(file) == 0 {
		return nil, errors.New("empty 'file' provided")
	}

	return &FileChecker{file: file}, nil
}

func FileCheckerFromMap(args map[string]any) (*FileChecker, error) {
	if args == nil {
		return nil, errors.New("empty args supplied")
	}

	file, ok := args["file"]
	if !ok {
		return nil, errors.New("no 'file' supplied")
	}

	return NewFileChecker(fmt.Sprintf("%s", file))
}

func (c *FileChecker) Name() string {
	return fmt.Sprintf("%s://%s", FileCheckerName, c.file)
}

func (c *FileChecker) IsHealthy(ctx context.Context) (bool, error) {
	_, err := os.Stat(c.file)
	if err == nil {
		return !c.wantsAbsence, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return c.wantsAbsence, nil
	}

	return false, err
}
