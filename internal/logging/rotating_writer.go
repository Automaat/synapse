package logging

import (
	"fmt"
	"os"
	"sync"
)

type RotatingWriter struct {
	mu       sync.Mutex
	file     *os.File
	path     string
	maxSize  int64
	maxFiles int
	size     int64
}

func NewRotatingWriter(path string, maxSize int64, maxFiles int) (*RotatingWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	return &RotatingWriter{
		file:     f,
		path:     path,
		maxSize:  maxSize,
		maxFiles: maxFiles,
		size:     info.Size(),
	}, nil
}

func (w *RotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

func (w *RotatingWriter) rotate() error {
	_ = w.file.Close()

	// shift existing rotated files: .N-1 -> .N, delete oldest
	for i := w.maxFiles - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", w.path, i)
		dst := fmt.Sprintf("%s.%d", w.path, i+1)
		if i == w.maxFiles-1 {
			_ = os.Remove(src)
		} else {
			_ = os.Rename(src, dst)
		}
	}

	// current -> .1
	_ = os.Rename(w.path, fmt.Sprintf("%s.1", w.path))

	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	w.size = 0
	return nil
}
