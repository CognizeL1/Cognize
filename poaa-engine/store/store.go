package store

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/hashicorp/golang-lru/v2"

	"github.com/cognize/axon/poaa-engine/dag"
)

var (
	PrefixVertex    = []byte("v:")
	PrefixPending   = []byte("p:")
	PrefixConfState = []byte("c:")
	PrefixTips      = []byte("t:")
	PrefixDepth     = []byte("d:")
	PrefixSender    = []byte("s:")
	PrefixArchive   = []byte("a:")
	PrefixMeta      = []byte("m:")
)

type DAGStore struct {
	db    *leveldb.DB
	cache *lru.Cache[[32]byte, *dag.Vertex]
	mu    sync.RWMutex
}

func NewDAGStore(path string) (*DAGStore, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	cache, err := lru.New[[32]byte, *dag.Vertex](10000)
	if err != nil {
		return nil, err
	}
	return &DAGStore{db: db, cache: cache}, nil
}

func (s *DAGStore) PutVertex(v *dag.Vertex) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := v.ToBytes()
	if err != nil {
		return err
	}

	batch := new(leveldb.Batch)

	key := append(PrefixVertex, v.Hash[:]...)
	batch.Put(key, data)

	depthKey := make([]byte, len(PrefixDepth)+8+32)
	copy(depthKey, PrefixDepth)
	binary.BigEndian.PutUint64(depthKey[len(PrefixDepth):], v.Depth)
	copy(depthKey[len(PrefixDepth)+8:], v.Hash[:])
	batch.Put(depthKey, v.Hash[:])

	senderKey := append(append(PrefixSender, []byte(v.Sender)...), v.Hash[:]...)
	batch.Put(senderKey, nil)

	idxKey := append(PrefixMeta, []byte("latest_index")...)
	idxVal := make([]byte, 8)
	binary.BigEndian.PutUint64(idxVal, v.Index)
	batch.Put(idxKey, idxVal)

	s.cache.Add(v.Hash, v)

	return s.db.Write(batch, nil)
}

func (s *DAGStore) GetVertex(hash [32]byte) (*dag.Vertex, error) {
	if v, ok := s.cache.Get(hash); ok {
		return v, nil
	}

	key := append(PrefixVertex, hash[:]...)
	data, err := s.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil, fmt.Errorf("vertex not found")
	}
	if err != nil {
		return nil, err
	}

	v := &dag.Vertex{}
	if err := v.FromBytes(data); err != nil {
		return nil, err
	}
	s.cache.Add(hash, v)
	return v, nil
}

func (s *DAGStore) HasVertex(hash [32]byte) bool {
	key := append(PrefixVertex, hash[:]...)
	exists, _ := s.db.Has(key, nil)
	return exists
}

func (s *DAGStore) GetLatestIndex() uint64 {
	key := append(PrefixMeta, []byte("latest_index")...)
	data, err := s.db.Get(key, nil)
	if err != nil {
		return 0
	}
	return binary.BigEndian.Uint64(data)
}

func (s *DAGStore) GetChildren(parentHash [32]byte) [][32]byte {
	prefix := append(PrefixVertex, parentHash[:]...)
	iter := s.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()

	var children [][32]byte
	for iter.Next() {
		key := iter.Key()
		if len(key) < 32 {
			continue
		}
		var child [32]byte
		copy(child[:], key[len(key)-32:])
		children = append(children, child)
	}
	return children
}

func (s *DAGStore) Close() {
	s.db.Close()
}
