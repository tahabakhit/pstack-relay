package engine

import (
	"fmt"
	"os"
)

type limitedLog struct {
	file *os.File
	left int64
}

func (l *limitedLog) Write(data []byte) (int, error) {
	if int64(len(data)) > l.left {
		n, err := l.file.Write(data[:int(l.left)])
		l.left -= int64(n)
		if err != nil {
			return n, err
		}
		return n, fmt.Errorf("worker log exceeds 32 MiB limit")
	}
	n, err := l.file.Write(data)
	l.left -= int64(n)
	return n, err
}
