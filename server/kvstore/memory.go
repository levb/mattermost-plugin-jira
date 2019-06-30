// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

type memoryStore struct {
	m map[string][]byte
}

var _ KVStore = (*memoryStore)(nil)

// NewMemoryStore creates a test store, backed by a map
func NewMemoryStore() KVStore {
	return &memoryStore{
		m: map[string][]byte{},
	}
}

func (s memoryStore) Load(key string) ([]byte, error) {
	data := s.m[key]
	if data == nil {
		return nil, ErrNotFound
	}
	return data, nil
}

func (s memoryStore) Store(key string, data []byte) error {
	s.m[key] = data
	return nil
}

func (s memoryStore) Delete(key string) error {
	delete(s.m, key)
	return nil
}

func (s memoryStore) Ensure(key string, newValue []byte) ([]byte, error) {
	if s.m[key] != nil {
		return s.m[key], nil
	}
	s.m[key] = newValue
	return newValue, nil
}
