/*
This Source Code Form is subject to the terms of the Mozilla
Public License, v. 2.0. If a copy of the MPL was not distributed
with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package ecs

import (
	"github.com/negrel/assert"
)

const (
	page_size_shift int32 = 10
	page_size       int32 = 1 << page_size_shift
	book_size       int32 = 1 << 10
)

type MapPage[K EntityID, V any] map[K]V
type PagedMap[K EntityID, V any] struct {
	len  int32
	book []MapPage[K, V]
}

func NewPagedMap[K EntityID, V any]() *PagedMap[K, V] {
	return &PagedMap[K, V]{
		book: make([]MapPage[K, V], book_size),
	}
}

func (m *PagedMap[K, V]) Get(key K) (value V, ok bool) {
	page_id := m.getPageId(key)
	if page_id >= cap(m.book) {
		return value, false
	}
	page := m.book[page_id]
	if page == nil {
		return value, false
	}
	v, ok := page[key]
	return v, ok
}

func (m *PagedMap[K, V]) Set(key K, value V) {
	page_id := m.getPageId(key)
	if page_id >= cap(m.book) {
		// extend the pages slice
		new_pages := make([]MapPage[K, V], cap(m.book)*2)
		m.book = append(m.book, new_pages...)
	}
	page := m.book[page_id]
	if page == nil {
		page = make(MapPage[K, V], page_size)
		m.book[page_id] = page
	}
	_, ok := page[key]
	if !ok {
		m.len++
	}
	page[key] = value
}

func (m *PagedMap[K, V]) Delete(key K) {
	page_id := m.getPageId(key)
	// Do not attempt to delete a value that does not exist
	assert.True(page_id < cap(m.book))
	page := m.book[page_id]
	// Do not attempt to delete a value that does not exist
	assert.True(page != nil)
	delete(page, key)
	m.len--
}

func (m *PagedMap[K, V]) getPageId(key K) int {
	id := key >> page_size_shift
	return int(id)
}

func (m *PagedMap[K, V]) Len() int32 {
	return m.len
}
