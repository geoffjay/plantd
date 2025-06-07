//go:build !integration
// +build !integration

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockMdpClient struct {
	mock.Mock
}

func newMockMdpClient() *mockMdpClient {
	return &mockMdpClient{}
}

func (c *mockMdpClient) Close() error {
	args := c.Called()
	return args.Error(0)
}

func (c *mockMdpClient) Send(service string, request ...string) (err error) {
	args := c.Called(service, request)
	return args.Error(0)
}

func (c *mockMdpClient) Recv() (reply []string, err error) {
	args := c.Called()
	return args.Get(0).([]string), args.Error(1)
}

// nolint: funlen
func TestClient(t *testing.T) {
	mockConn := newMockMdpClient()

	mockConn.On(
		"Send",
		"org.plantd.module.Echo",
		[]string{"echo", `{"message":"foo","service":"org.plantd.Client"}`},
	).Return(nil).Once()
	mockConn.On("Recv").Return(
		[]string{`{"message": "foo", "service": "org.plantd.Client"}`}, nil,
	).Once()

	// Test NewClient
	client := &Client{conn: mockConn}
	assert.NotNil(t, client)

	// Test Client.SendRawRequest
	request := &RawRequest{
		"service": "org.plantd.Client",
		"message": "foo",
	}
	resp, err := client.SendRawRequest("org.plantd.module.Echo", "echo", request)

	assert.Nil(t, err)
	assert.Equal(t, "foo", resp["message"])

	mockConn.AssertCalled(
		t,
		"Send",
		"org.plantd.module.Echo",
		[]string{"echo", `{"message":"foo","service":"org.plantd.Client"}`},
	)
	mockConn.AssertCalled(t, "Recv")
}
