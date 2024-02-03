# Raft-Example
This is a simple key-value store that is replicated using [jmsadair/raft](https://github.com/jmsadair/raft). This is a minimal implementation that is intended to serve as an example of how one might use raft in a real system as well as for testing purposes. 
It should not be used in production.

## Usage
To run a demonstration of this repository, just run 

```
bash demo.sh
```

This will execute a script that starts a cluster of five nodes, submits some operations, and crashes some of the nodes. You can also provide
some arguments to `demo.sh` to modify the demonstration. For example, you can specify the number of clients and the number of operations each
client should submit as so:

```
bash demo.sh -c 5 -o 500
```

This will again spin up a cluster of five nodes; however, there will now be 5 clients concurrently submitting 500 operations each. It's also
possible to generate a Jepsen log which can be used to verify the [linearizability](https://en.wikipedia.org/wiki/Linearizability) of the operations (take a look at [Knossos](https://github.com/jepsen-io/knossos)) by passing the `--history` flag.

```
bash demo.sh -c 5 -o 500 --history
```
Refer to `demo.sh` to see all options.

## Testing
This repository also includes a Jepsen test for the key-value store. To run the test, you are going to need to setup a Jepsen environment, including a control node with a JVM and 
Leiningen, and a collection of Debian nodes to install this example on. There are instructions for setting up a Jepsen environent [here](https://github.com/jepsen-io/jepsen).

Once your environment is ready, you should be able to run something like:

```
lein run test --test-count 10
```

This runs a test that randomly partitions the cluster every few seconds while operations are being submitted. Once complete, the test will verify that the history is [linearizable](https://en.wikipedia.org/wiki/Linearizability).
