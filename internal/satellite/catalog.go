package satellite

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"orbitlink/internal/store"
)

type Catalog struct {
	mu    sync.RWMutex
	path  string
	items map[string]Satellite
}

func NewCatalog(paths store.Paths) (*Catalog, error) {
	catalog := &Catalog{path: paths.Snapshot("satellites"), items: make(map[string]Satellite)}
	var items []Satellite
	found, err := store.ReadJSON(catalog.path, &items)
	if err != nil {
		return nil, err
	}
	if found {
		for _, item := range items {
			catalog.items[item.ID] = item
		}
	}
	return catalog, nil
}

func (c *Catalog) Upsert(value Satellite) error {
	if err := value.Validate(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	value.Updated = time.Now().UTC()
	c.items[value.ID] = value
	return c.flushLocked()
}

func (c *Catalog) Get(id string) (Satellite, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, found := c.items[id]
	if !found {
		return Satellite{}, fmt.Errorf("satellite %s not found", id)
	}
	return value, nil
}

func (c *Catalog) List() []Satellite {
	c.mu.RLock()
	defer c.mu.RUnlock()
	values := make([]Satellite, 0, len(c.items))
	for _, value := range c.items {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return values
}

func (c *Catalog) flushLocked() error {
	values := make([]Satellite, 0, len(c.items))
	for _, value := range c.items {
		values = append(values, value)
	}
	return store.WriteJSON(c.path, values)
}
