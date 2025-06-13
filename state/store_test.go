package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type StoreTestSuite struct {
	suite.Suite
	scope string
	store *Store
}

func TestStoreLoad(t *testing.T) {
	store := NewStore()
	if err := os.Mkdir("./tmp", 0664); err != nil {
		panic(err)
	}
	err := store.Load("./tmp")
	assert.Error(t, err, "should not be able to load a directory")
	_ = os.Remove("./tmp")
}

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}

func (suite *StoreTestSuite) SetupSuite() {
	suite.scope = "org.plantd.State.Test"
	suite.store = NewStore()
	if err := suite.store.Load("/tmp/test.db"); err != nil {
		panic(err)
	}
	if err := suite.store.CreateScope(suite.scope); err != nil {
		panic(err)
	}
}

func (suite *StoreTestSuite) TearDownSuite() {
	if err := suite.store.DeleteScope(suite.scope); err != nil {
		panic(err)
	}
	suite.store.Unload()
}

// nolint: typecheck
func (suite *StoreTestSuite) TestStore_GetMissingKey() {
	value, err := suite.store.Get("org.plantd.State.Test", "missing")
	suite.NoError(err, err)
	suite.Equal(value, "")
}

// nolint: typecheck
func (suite *StoreTestSuite) TestStore_SetGet() {
	err := suite.store.Set("org.plantd.State.Test", "foo", "bar")
	suite.NoError(err, err)
	value, err := suite.store.Get("org.plantd.State.Test", "foo")
	suite.NoError(err, err)
	suite.Equal(value, "bar")
}

func (suite *StoreTestSuite) TestStore_Delete() {
	err := suite.store.Set("org.plantd.State.Test", "foo", "bar")
	suite.NoError(err, err)
	err = suite.store.Delete("org.plantd.State.Test", "foo")
	suite.NoError(err, err)
	value, err := suite.store.Get("org.plantd.State.Test", "foo")
	suite.NoError(err, err)
	suite.Equal(value, "")
}

// nolint: typecheck
func (suite *StoreTestSuite) TestStore_Scope() {
	var err error
	err = suite.store.CreateScope("test")
	suite.NoError(err, err)
	err = suite.store.DeleteScope("fake")
	suite.Error(err, "should not be able to delete a non existing scope")
	err = suite.store.DeleteScope("test")
	suite.NoError(err, err)
}

func (suite *StoreTestSuite) TestStore_HasScope() {
	err := suite.store.CreateScope("test")
	suite.NoError(err, err)
	ok := suite.store.HasScope("test")
	suite.Equal(ok, true)
	err = suite.store.DeleteScope("test")
	suite.NoError(err, err)
}

func (suite *StoreTestSuite) TestStore_ListAllScope() {
	var err error
	err = suite.store.CreateScope("test1")
	suite.NoError(err, err)
	err = suite.store.CreateScope("test2")
	suite.NoError(err, err)
	scopes := suite.store.ListAllScope()
	suite.Equal(len(scopes), 3)
	suite.Equal(scopes[1], "test1")
	suite.Equal(scopes[2], "test2")
	err = suite.store.DeleteScope("test1")
	suite.NoError(err, err)
	err = suite.store.DeleteScope("test2")
	suite.NoError(err, err)
}

func (suite *StoreTestSuite) TestStore_ListAllKeys() {
	var err error

	// Create a scope and add some keys
	err = suite.store.CreateScope("testscope")
	suite.NoError(err, err)

	// Add some test data
	err = suite.store.Set("testscope", "key1", "value1")
	suite.NoError(err, err)
	err = suite.store.Set("testscope", "key2", "value2")
	suite.NoError(err, err)
	err = suite.store.Set("testscope", "key3", "value3")
	suite.NoError(err, err)

	// List all keys
	keys, err := suite.store.ListAllKeys("testscope")
	suite.NoError(err, err)
	suite.Equal(len(keys), 3)
	suite.Contains(keys, "key1")
	suite.Contains(keys, "key2")
	suite.Contains(keys, "key3")

	// Test non-existent scope
	_, err = suite.store.ListAllKeys("nonexistent")
	suite.Error(err, "should error for non-existent scope")

	// Cleanup
	err = suite.store.DeleteScope("testscope")
	suite.NoError(err, err)
}

func (suite *StoreTestSuite) TestStore_ListAllKeysWithValues() {
	var err error

	// Create a scope and add some keys
	err = suite.store.CreateScope("testscope")
	suite.NoError(err, err)

	// Add some test data
	err = suite.store.Set("testscope", "key1", "value1")
	suite.NoError(err, err)
	err = suite.store.Set("testscope", "key2", "value2")
	suite.NoError(err, err)

	// List all keys with values
	data, err := suite.store.ListAllKeysWithValues("testscope")
	suite.NoError(err, err)
	suite.Equal(len(data), 2)
	suite.Equal(data["key1"], "value1")
	suite.Equal(data["key2"], "value2")

	// Test non-existent scope
	_, err = suite.store.ListAllKeysWithValues("nonexistent")
	suite.Error(err, "should error for non-existent scope")

	// Cleanup
	err = suite.store.DeleteScope("testscope")
	suite.NoError(err, err)
}
