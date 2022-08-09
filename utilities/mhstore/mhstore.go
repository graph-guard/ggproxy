package mhstore

import "github.com/graph-guard/gguard-proxy/utilities/aset"

// MHStore stands for Mask-Hash-Store.
// Storage for faster calculation of matching hashes per mask.
type MHStore struct {
	store [65536][]uint64
	set   []uint16
}

// New creates a new instance of MHStore.
func New() *MHStore {
	return &MHStore{
		set: []uint16{},
	}
}

// Reset resets the storage.
func (s *MHStore) Reset() {
	for _, hash := range s.set {
		s.store[hash] = s.store[hash][:0]
	}
	s.set = s.set[:0]
}

// Add adds a hash to the storage at corresponding mask.
func (s *MHStore) Add(mask uint16, hash uint64) {
	var i int
	var found bool
	if len(s.store[mask]) >= 256 {
		i, found = aset.FindExp(s.store[mask], hash)
	} else {
		i, found = aset.FindBin(s.store[mask], hash, 0, len(s.store[mask])-1)
	}
	if !found {
		if i == len(s.store[mask]) {
			s.store[mask] = append(s.store[mask], hash)
		} else {
			s.store[mask] = append(s.store[mask][:i+1], s.store[mask][i:]...)
			s.store[mask][i] = hash
		}
	}

	if len(s.set) >= 256 {
		i, found = aset.FindExp(s.set, mask)
	} else {
		i, found = aset.FindBin(s.set, mask, 0, len(s.set)-1)
	}
	if !found {
		if i == len(s.set) {
			s.set = append(s.set, mask)
		} else {
			s.set = append(s.set[:i+1], s.set[i:]...)
			s.set[i] = mask
		}
	}
}

// Get returns a hash array at the corresponding mask.
func (s *MHStore) Get(mask uint16) []uint64 {
	return s.store[mask]
}

// Len returns the storage length.
func (s *MHStore) Len() int {
	return len(s.set)
}
