package common

import (
	"sync"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

// Result fills conn.IterResult interface for all sql dbs
type Result struct {
	next     func() (models.Row, error)
	header   models.Header
	close    func()
	meta     models.Meta
	callback func()
	once     sync.Once
}

func (r *Result) SetCustomHeader(header models.Header) {
	r.header = header
}

func (r *Result) SetCallback(callback func()) {
	r.callback = callback
}

func (r *Result) Meta() (models.Meta, error) {
	return r.meta, nil
}

func (r *Result) Header() (models.Header, error) {
	return r.header, nil
}

func (r *Result) Next() (models.Row, error) {
	rows, err := r.next()
	if err != nil || rows == nil {
		r.Close()
		return nil, err
	}
	return rows, nil
}

func (r *Result) Close() {
	r.close()
	if r.callback != nil {
		r.once.Do(r.callback)
	}
}

// ResultBuilder builds the rows
type ResultBuilder struct {
	next   func() (models.Row, error)
	header models.Header
	close  func()
	meta   models.Meta
}

func NewResultBuilder() *ResultBuilder {
	return &ResultBuilder{
		next:   func() (models.Row, error) { return nil, nil },
		header: models.Header{},
		close:  func() {},
		meta:   models.Meta{},
	}
}

func (b *ResultBuilder) WithNextFunc(fn func() (models.Row, error)) *ResultBuilder {
	b.next = fn
	return b
}

func (b *ResultBuilder) WithHeader(header models.Header) *ResultBuilder {
	b.header = header
	return b
}

func (b *ResultBuilder) WithCloseFunc(fn func()) *ResultBuilder {
	b.close = fn
	return b
}

func (b *ResultBuilder) WithMeta(meta models.Meta) *ResultBuilder {
	b.meta = meta
	return b
}

func (b *ResultBuilder) Build() *Result {
	return &Result{
		next:   b.next,
		header: b.header,
		close:  b.close,
		meta:   b.meta,
		once:   sync.Once{},
	}
}
