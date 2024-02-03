package server

import (
	"bytes"
	"io"
	"log"
	"sync"

	pb "github.com/jmsadair/raft-example/api"

	"github.com/jmsadair/raft"
	"google.golang.org/protobuf/proto"
)

// The number of log entries that will trigger a snapshot.
const snapshotSize = 1000

// KeyValueStore is a data structure that stores key-value pairs.
type KeyValueStore struct {
	// A table that maps key to value.
	keyValueTable map[string]string

	// A table mapping client ID to a session.
	sessionTable map[uint32]*pb.Session

	mu sync.Mutex
}

// NewKeyValueStore creates a new instance of a key-value store.
func NewKeyValueStore() *KeyValueStore {
	return &KeyValueStore{
		keyValueTable: make(map[string]string),
		sessionTable:  make(map[uint32]*pb.Session),
	}
}

// Apply applies an operation to the key-value store.
func (kv *KeyValueStore) Apply(operation *raft.Operation) interface{} {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	request := &pb.Request{}
	if err := proto.Unmarshal(operation.Bytes, request); err != nil {
		log.Fatalf("failed to unmarshal operation: error = %v", err)
	}

	// Make sure this is not a request that has already been applied.
	session, ok := kv.sessionTable[request.GetClient()]
	if ok && session.GetSequenceNumber() >= request.GetSequenceNumber() {
		return &pb.Response{Value: session.GetLastResult()}
	}

	// Create a session if this client is new.
	if !ok {
		session = &pb.Session{}
		kv.sessionTable[request.GetClient()] = session
	}

	// Apply the operation to the store.
	// This is a simple implementation, so we just assume that all replicated operations are writes.
	var value string
	if operation.OperationType == raft.Replicated {
		value = kv.Put(request.GetKey(), request.GetValue())
	} else {
		value = kv.Get(request.GetKey())
	}

	// Update the client session.
	session.SequenceNumber = request.GetSequenceNumber()
	session.LastResult = value

	return &pb.Response{Value: value}
}

// Put sets the value of a key and returns the value it was set to.
func (kv *KeyValueStore) Put(key string, value string) string {
	kv.keyValueTable[key] = value
	return value
}

// Get returns the value of a key. If the key does not exist, an
// empty string is returned.
func (kv *KeyValueStore) Get(key string) string {
	return kv.keyValueTable[key]
}

// Restore restores the key-value store from a snapshot taken by raft.
func (kv *KeyValueStore) Restore(snapshotReader io.Reader) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	protoKeyValueStore := &pb.KeyValueStore{}
	data, err := io.ReadAll(snapshotReader)
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(data, protoKeyValueStore); err != nil {
		return err
	}

	kv.keyValueTable = protoKeyValueStore.GetKeyValueTable()
	kv.sessionTable = protoKeyValueStore.GetSessionTable()

	return nil
}

// Snapshot writes the state of the key-value store to a snapshot file.
func (kv *KeyValueStore) Snapshot(snapshotWriter io.Writer) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	protoKeyValueStore := &pb.KeyValueStore{
		KeyValueTable: kv.keyValueTable,
		SessionTable:  kv.sessionTable,
	}

	data, err := proto.Marshal(protoKeyValueStore)
	if err != nil {
		return err
	}
	dataReader := bytes.NewReader(data)
	if _, err := io.Copy(snapshotWriter, dataReader); err != nil {
		return err
	}

	return nil
}

// NeedSnapshot returns true if a snapshot should be taken of
// the key-value store and false otherwise.
func (kv *KeyValueStore) NeedSnapshot(logSize int) bool {
	return logSize >= snapshotSize
}
