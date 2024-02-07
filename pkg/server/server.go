package server

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/jmsadair/raft-example/api"

	"github.com/jmsadair/raft"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	// ErrNotLeader is returned by a server when an operation is
	// submitted to it but it is not the leader. The client should
	// try submitting the operation to a different server.
	ErrNotLeader = status.Error(
		codes.FailedPrecondition,
		"keyValueServer: this node is not the leader",
	)

	// ErrTimeout is returned when a submitted operation times out.
	// This typically occurs if there are network partitions. The
	// client should try submitting the operation to a different server.
	ErrTimeout = status.Error(
		codes.Unavailable,
		"keyValueServer: the submitted operation timed out",
	)
)

// futureTimeout is the maximum amount of time that an operation will
// wait for a response before timing out.
const futureTimeout = 500 * time.Millisecond

// Server is a simple key-value server that is replicated using raft.
type Server struct {
	pb.UnimplementedKeyValueServer

	// The base server implementation.
	server *grpc.Server

	// The ID of this server.
	id string

	// The listen address of this server.
	address net.Addr

	// The raft node that is used to replicate operations.
	node *raft.Raft
}

// NewServer creates a new Server instance with the provided ID, address, and raft node.
func NewServer(id string, address string, node *raft.Raft) (*Server, error) {
	resolvedAddress, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}
	return &Server{node: node, address: resolvedAddress, id: id}, nil
}

// Start starts the server.
func (s *Server) Start() error {
	if err := s.node.Start(); err != nil {
		return err
	}
	listener, err := net.Listen(s.address.Network(), s.address.String())
	if err != nil {
		return err
	}
	s.server = grpc.NewServer()
	pb.RegisterKeyValueServer(s.server, s)
	go s.server.Serve(listener)
	return nil
}

// Stop stops the server.
func (s *Server) Stop() {
	defer s.node.Stop()
	s.server.Stop()
}

// Get retreives the value of a key.
func (s *Server) Get(ctx context.Context, request *pb.Request) (*pb.Response, error) {
	return s.submitOperation(request, raft.LinearizableReadOnly)
}

// Put sets the value of a key.
func (s *Server) Put(ctx context.Context, request *pb.Request) (*pb.Response, error) {
	return s.submitOperation(request, raft.Replicated)
}

func (s *Server) submitOperation(
	request *pb.Request,
	operationType raft.OperationType,
) (*pb.Response, error) {
	operation, err := proto.Marshal(request)
	if err != nil {
		log.Fatalf("failed to marshal request: error = %v", err)
	}

	future := s.node.SubmitOperation(operation, operationType, futureTimeout)
	result := future.Await()
	if err := result.Error(); err != nil {
		switch err {
		case raft.ErrNotLeader:
			return nil, ErrNotLeader
		case raft.ErrTimeout:
			return nil, ErrTimeout
		default:
			log.Fatalf("unexpected error type: error = %v", err)
		}
	}

	response := result.Success()

	return response.ApplicationResponse.(*pb.Response), nil
}
