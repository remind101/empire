package stores

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
)

type Store interface {
	Get(k string, v interface{}) (bool, error)
	Set(k string, v interface{}) error
	List(k string, v interface{}) error
}

type memStore struct {
	sync.Mutex
	store map[string][]byte
}

func NewMemStore() *memStore {
	return &memStore{store: make(map[string][]byte)}
}

func (s *memStore) Get(k string, v interface{}) (bool, error) {
	s.Lock()
	defer s.Unlock()

	if b, found := s.store[k]; found {
		return found, json.Unmarshal(b, v)
	}

	return false, nil
}

func (s *memStore) Set(k string, v interface{}) error {
	s.Lock()
	defer s.Unlock()

	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	s.store[k] = b
	return nil
}

func (s *memStore) List(k string, v interface{}) error {
	s.Lock()
	defer s.Unlock()

	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Slice {
		return errors.New("v must be a pointer to a slice")
	}

	sliceValue := reflect.Indirect(reflect.ValueOf(v))

	// Get the element type of the slice
	elemType := reflect.TypeOf(v).Elem().Elem()

	pointerElements := elemType.Kind() == reflect.Ptr
	if pointerElements {
		elemType = elemType.Elem()
	}

	for key, b := range s.store {
		if !strings.HasPrefix(key, k) {
			continue
		}

		elem := reflect.New(elemType)

		if err := json.Unmarshal(b, elem.Interface()); err != nil {
			return err
		}
		if !pointerElements {
			elem = elem.Elem()
		}
		sliceValue.Set(reflect.Append(sliceValue, elem))
	}

	return nil
}
