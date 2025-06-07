package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HandlerTestSuite struct {
	suite.Suite
	handler *Handler
}

type testCallback struct {
	name string
}

func (cb *testCallback) Execute(data string) ([]byte, error) {
	return []byte(data), nil
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (suite *HandlerTestSuite) SetupSuite() {
	suite.handler = NewHandler()
	err := suite.handler.AddCallback("test", &testCallback{name: "test"})
	suite.NoError(err, "callback added successfully")
}

func (suite *HandlerTestSuite) TestHandler_Add() {
	err := suite.handler.AddCallback("test", &testCallback{name: "test"})
	suite.ErrorContains(err, "callback already exists")
	err = suite.handler.AddCallback("foo", &testCallback{name: "foo"})
	suite.NoError(err, "callback added successfully")
}

// nolint: typecheck
func (suite *HandlerTestSuite) TestHandler_Get() {
	callback, err := suite.handler.GetCallback("test")
	assert.NotNil(suite.T(), callback, "callback exists for `test`")
	assert.Nil(suite.T(), err, "callback retrieve failed")
	callback, err = suite.handler.GetCallback("foo-get")
	assert.Nil(suite.T(), callback, "callback doesn't exist for `foo-get`")
	assert.NotNil(suite.T(), err,
		"callback retrieval failed for unavailable name")
	assert.Equal(suite.T(), err.Error(), "callback not found for foo-get")
}
