package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

// Store type is used to access the on disk KV store.
type Store struct {
	db *bolt.DB
}

// NewStore constructs a new instance of a Store.
func NewStore() *Store {
	return &Store{}
}

// Load opens the KV store file at `path`.
func (s *Store) Load(path string) (err error) {
	if s.db, err = bolt.Open(path, 0664, nil); err != nil {
		return err
	}
	return nil
}

// Unload is used to close the database connection.
func (s *Store) Unload() {
	if err := s.db.Close(); err != nil {
		log.WithFields(log.Fields{
			"context": "store.unload",
			"error":   err,
		}).Error("failed to close database")
	}
}

// HasScope checks if the bucket with the name `scope` exists.
func (s *Store) HasScope(scope string) bool {
	exists := false
	_ = s.db.View(func(tx *bolt.Tx) error {
		if bucket := tx.Bucket([]byte(scope)); bucket != nil {
			exists = true
		}
		return nil
	})
	return exists
}

// CreateScope creates a new bucket in the store with the name `scope`.
func (s *Store) CreateScope(scope string) (err error) {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err = tx.CreateBucketIfNotExists([]byte(scope)); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteScope removes a bucket from the store with the name `scope`.
func (s *Store) DeleteScope(scope string) (err error) {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err = tx.DeleteBucket([]byte(scope)); err != nil {
		return err
	}

	return tx.Commit()
}

// ListAllScope returns a list of all scope names (bucket names) in the store.
func (s *Store) ListAllScope() (list []string) {
	_ = s.db.View(func(tx *bolt.Tx) error {
		_ = tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			list = append(list, string(name))
			return nil
		})
		return nil
	})
	return
}

// DebugScope prints the contents of a bucket using the debug log level.
func (s *Store) DebugScope(scope string) {
	_ = s.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		bucket := tx.Bucket([]byte(scope))
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			log.WithFields(log.Fields{"scope": scope}).Debugf(
				"key=%s, value=%s\n", k, v)
		}
		return nil
	})
}

// Get a value at `key` in the bucket named `scope`.
func (s *Store) Get(scope, key string) (value string, err error) {
	log.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Trace("KV get")
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(scope))
		if bucket == nil {
			return fmt.Errorf("scope `%s` doesn't exist", scope)
		}
		value = string(bucket.Get([]byte(key)))
		return nil
	})
	return
}

// Set `value` at `key` in the bucket named `scope`.
func (s *Store) Set(scope, key, value string) (err error) {
	log.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
		"value": value,
	}).Trace("KV set")
	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(scope))
		if bucket == nil {
			if err := s.CreateScope(scope); err != nil {
				return err
			}
		}
		err := bucket.Put([]byte(key), []byte(value))
		return err
	})
	return
}

// Delete `key` in the bucket named `scope`.
func (s *Store) Delete(scope, key string) (err error) {
	log.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Trace("KV delete")
	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(scope))
		if bucket == nil {
			return fmt.Errorf("scope `%s` doesn't exist", scope)
		}
		err := bucket.Delete([]byte(key))
		return err
	})
	return
}
