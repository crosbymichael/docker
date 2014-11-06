package multiwriter

import (
	"io"
	"sync"
)

type MultiWriter struct {
	sync.RWMutex

	writers []io.WriteCloser
}

func NewMultiWriter() *MultiWriter {
	return &MultiWriter{}
}

func (m *MultiWriter) Write(p []byte) (n int, err error) {
	m.RLock()
	defer m.RUnlock()

	plen := len(p)

	for _, w := range m.writers {
		if n, err = w.Write(p); err != nil {
			return n, err
		}

		if n != plen {
			return n, io.ErrShortWrite
		}
	}

	return plen, nil
}

func (m *MultiWriter) Add(w io.WriteCloser) {
	m.Lock()
	m.writers = append(m.writers, w)
	m.Unlock()
}

func (m *MultiWriter) Close() (err error) {
	m.Lock()

	for _, w := range m.writers {
		if nerr := w.Close(); err == nil {
			err = nerr
		}
	}

	m.writers = nil

	m.Unlock()

	return err
}

// Clean closes and removes all writers. Last non-eol-terminated part of data
// will be saved.
func (m *MultiWriter) Clean() error {
	m.Lock()
	for _, w := range m.writers {
		w.Close()
	}
	m.writers = nil
	m.Unlock()
	return nil
}
