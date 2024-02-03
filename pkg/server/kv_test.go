package server

import (
	"bytes"
	"testing"

	"github.com/jmsadair/raft"
	pb "github.com/jmsadair/raft-example/api"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestPutGet(t *testing.T) {
	kv := NewKeyValueStore()

	key := "x"
	value := "y"
	require.Equal(t, value, kv.Put(key, value))
	require.Equal(t, value, kv.Get(key))
}

func TestApply(t *testing.T) {
	kv := NewKeyValueStore()

	// Apply a put operation.
	command := &pb.Request{Key: "x", Value: "y", Client: 1, SequenceNumber: 1}
	commandBytes, err := proto.Marshal(command)
	require.NoError(t, err)
	operation := raft.Operation{
		OperationType: raft.Replicated,
		Bytes:         commandBytes,
	}
	result := kv.Apply(&operation)
	response, ok := result.(*pb.Response)
	require.True(t, ok)
	require.Equal(t, command.Value, response.Value)

	// Apply a get operation.
	command.SequenceNumber = 2
	commandBytes, err = proto.Marshal(command)
	require.NoError(t, err)
	operation.OperationType = raft.LinearizableReadOnly
	operation.Bytes = commandBytes
	result = kv.Apply(&operation)
	response, ok = result.(*pb.Response)
	require.True(t, ok)
	require.Equal(t, "y", response.Value)

	// Make sure duplicates are handled.
	command.Key = ""
	command.Value = ""
	commandBytes, err = proto.Marshal(command)
	require.NoError(t, err)
	operation.Bytes = commandBytes
	result = kv.Apply(&operation)
	response, ok = result.(*pb.Response)
	require.True(t, ok)
	require.Equal(t, "y", response.Value)
}

func TestSnapshotRestore(t *testing.T) {
	kv := NewKeyValueStore()

	keyValueTable := map[string]string{"x": "y"}
	sessionTable := map[uint32]*pb.Session{1: {SequenceNumber: 1, LastResult: "z"}}
	kv.keyValueTable = keyValueTable
	kv.sessionTable = sessionTable

	// Take a snapshot of the key-value store.
	snapshotWriter := new(bytes.Buffer)
	err := kv.Snapshot(snapshotWriter)
	require.NoError(t, err)

	// Restore a new key-value store with the snapshot.
	kv = NewKeyValueStore()
	err = kv.Restore(snapshotWriter)
	require.NoError(t, err)
	require.Equal(t, keyValueTable, kv.keyValueTable)

	// Unfortunately testify does not support protobuf message equality.
	session, ok := kv.sessionTable[1]
	require.True(t, ok)
	require.True(t, proto.Equal(session, sessionTable[1]))
}
