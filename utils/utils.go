package utils

import (
	"reflect"
	"slices"
	"sync"

	"github.com/wtsi-hgi/softpack/db"
)

type WaitingEnvs struct {
	mu sync.RWMutex

	m map[db.RecipeRequest][]db.Environment
}

func New() WaitingEnvs {
	return WaitingEnvs{
		m: make(map[db.RecipeRequest][]db.Environment),
	}
}

func (m *WaitingEnvs) Get(key db.RecipeRequest) ([]db.Environment, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.m[key]

	return slices.Clone(value), ok
}

func (m *WaitingEnvs) Append(key db.RecipeRequest, value db.Environment) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.m[key] = append(m.m[key], value)
}

func (m *WaitingEnvs) Delete(key db.RecipeRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.m, key)
}

func (m *WaitingEnvs) ContainsEnv(e db.Environment) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, envs := range m.m {
		for _, env := range envs {
			if reflect.DeepEqual(e, env) {
				return true
			}
		}
	}

	return false
}
