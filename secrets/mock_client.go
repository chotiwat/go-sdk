package secrets

import (
	"context"
	"fmt"
)

var _ Client = &MockClient{}

// NewMockClient creates a new mock Client.
func NewMockClient() *MockClient {
	return &MockClient{
		SecretValues: make(map[string]Values),
	}
}

// MockClient is a mock events Client
type MockClient struct {
	SecretValues map[string]Values
}

// Put puts a value.
func (c *MockClient) Put(_ context.Context, key string, data Values, options ...Option) error {
	c.SecretValues[key] = data

	return nil
}

// Get gets a value at a given key.
func (c *MockClient) Get(_ context.Context, key string, options ...Option) (Values, error) {
	val, exists := c.SecretValues[key]
	if !exists {
		return nil, fmt.Errorf("Key not found: %s", key)
	}

	return val, nil
}

// Delete deletes a key.
func (c *MockClient) Delete(_ context.Context, key string, options ...Option) error {
	if _, exists := c.SecretValues[key]; !exists {
		return fmt.Errorf("Key not found: %s", key)
	}

	delete(c.SecretValues, key)
	return nil
}
