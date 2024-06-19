package ttrpcutil

import (
	"context"
	"demo/over/dialer"
	"errors"
	"fmt"
	"sync"
	"time"

	v1 "demo/over/api/services/ttrpc/events/v1"
	"demo/over/ttrpc"
)

const ttrpcDialTimeout = 5 * time.Second

type ttrpcConnector func() (*ttrpc.Client, error)

// Client is the client to interact with TTRPC part of containerd server (plugins, events)
type Client struct {
	mu        sync.Mutex
	connector ttrpcConnector
	client    *ttrpc.Client
	closed    bool
}

// NewClient returns a new containerd TTRPC client that is connected to the containerd instance provided by address
func NewClient(address string, opts ...ttrpc.ClientOpts) (*Client, error) {
	connector := func() (*ttrpc.Client, error) {
		ctx, cancel := context.WithTimeout(context.Background(), ttrpcDialTimeout)
		defer cancel()
		conn, err := dialer.ContextDialer(ctx, address)
		if err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}

		client := ttrpc.NewClient(conn, opts...)
		return client, nil
	}

	return &Client{
		connector: connector,
	}, nil
}

// Reconnect re-establishes the TTRPC connection to the containerd daemon
func (c *Client) Reconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connector == nil {
		return errors.New("unable to reconnect to containerd, no connector available")
	}

	if c.closed {
		return errors.New("client is closed")
	}

	if c.client != nil {
		if err := c.client.Close(); err != nil {
			return err
		}
	}

	client, err := c.connector()
	if err != nil {
		return err
	}

	c.client = client
	return nil
}

// EventsService creates an EventsService client
func (c *Client) EventsService() (v1.EventsService, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return v1.NewEventsClient(client), nil
}

// Client returns the underlying TTRPC client object
func (c *Client) Client() (*ttrpc.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == nil {
		client, err := c.connector()
		if err != nil {
			return nil, err
		}
		c.client = client
	}
	return c.client, nil
}

// Close closes the clients TTRPC connection to containerd
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
