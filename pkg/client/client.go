package client

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	pb "github.com/jmsadair/raft-example/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ErrTimeout is an error returned when the client is unable to succesfully
// complete the operation within the specified timeout.
var ErrTimeout = errors.New("operation failed: client-specified timeout elapsed")

// Client is a client of the key-value store.
type Client struct {
	// The unique id of this client.
	id uuid.UUID

	// The current sequence number of this client. This is used
	// by the server to deduplicate requests from this client.
	sequenceNumber uint64

	// The ID that this client knows to be the leader.
	leaderID string

	// The ID and address of all the key-value servers.
	nodes map[string]string

	// Actual client implementation.
	clients map[string]pb.KeyValueClient
}

// NewClient creates a new instance of a client.
func NewClient(nodes map[string]string) (*Client, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	clients := make(map[string]pb.KeyValueClient, len(nodes))
	for id, address := range nodes {
		conn, err := grpc.Dial(
			address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, err
		}
		clients[id] = pb.NewKeyValueClient(conn)
	}

	return &Client{
		id:      id,
		nodes:   nodes,
		clients: clients,
	}, nil
}

// Get gets the value associated with the provided key and returns it.
// An error is returned if the client is unable to complete the operation within the provided timeout.
func (c *Client) Get(key string, timeout time.Duration) (string, error) {
	request := &pb.Request{Key: key, Client: c.id.ID(), SequenceNumber: c.sequenceNumber}

	submit := func(c pb.KeyValueClient, request *pb.Request) (*pb.Response, error) {
		return c.Get(context.Background(), request)
	}

	return c.submitOperation(request, timeout, submit)
}

// Put sets the value of the provided key.
// An error is returned if the client is unable to complete the operation within the provided timeout.
// Note that, if the operation times out, the operation may or may not have been executed.
func (c *Client) Put(key string, value string, timeout time.Duration) (string, error) {
	request := &pb.Request{
		Key:            key,
		Value:          value,
		Client:         c.id.ID(),
		SequenceNumber: c.sequenceNumber,
	}

	submit := func(c pb.KeyValueClient, request *pb.Request) (*pb.Response, error) {
		return c.Put(context.Background(), request)
	}

	return c.submitOperation(request, timeout, submit)
}

func (c *Client) submitOperation(
	request *pb.Request,
	timeout time.Duration,
	submit func(pb.KeyValueClient, *pb.Request) (*pb.Response, error),
) (string, error) {
	defer func() {
		c.sequenceNumber++
	}()

	for start := time.Now(); time.Since(start) < timeout; {
		// If there is a cached leader, try sending the operation to that server first.
		if c.leaderID != "" {
			client := c.clients[c.leaderID]
			response, err := submit(client, request)
			if err == nil {
				return response.Value, nil
			}
			c.leaderID = ""
		}

		// Otherwise, try the other servers until the operation is successful.
		for id, client := range c.clients {
			response, err := submit(client, request)
			if err == nil {
				c.leaderID = id
				return response.Value, nil
			}
		}
	}

	return "", ErrTimeout
}
