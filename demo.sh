#!/bin/bash

set -e

# The IDs and addresses of the raft servers.
cluster="n1:127.0.0.1:8080,n2:127.0.0.2:8080,n3:127.0.0.3:8080,n4:127.0.0.4:8080,n5:127.0.0.5:8080"

# The IDs and address of the key-value servers.
kv_servers="n1:127.0.0.1:8081,n2:127.0.0.2:8081,n3:127.0.0.3:8081,n4:127.0.0.4:8081,n5:127.0.0.5:8081"

# Number of clients.
num_client_processes=1

# Number of operations each client will submit.
num_operations=500

# Indicates whether the script should generate a Jepsen log.
generate_history=false

# Indicates whether the script should remove server data upon completion.
remove_data=false

# If history is generated, it will be written to this file.
jepsen_log="history.edn"

# Error returned when operation times out.
timeout_pattern="operation failed: client-specified timeout elapsed"

# Kills all server processes.
function kill_servers {
	pkill -f "kv-server"
}

# Indicates whether the client processes have submitted all operations.
function clients_running {
	for pid in "${client_pids[@]}"; do
		if ps -p "$pid" >/dev/null 2>&1; then
			return 0
		fi
	done
	return 1
}

# Function to display usage information
usage() {
	echo "Usage: $0 [-c <num_client_processes>] [-o <num_operations>] [--history] [--help]"
	echo "Options:"
	echo "  -c <num_client_processes>: Number of client processes (default: 1)"
	echo "  -o <num_operations>: Number of operations submitted by each client (default: 200)"
	echo "  --clean: Delete server data upon completion (default: false)"
	echo "  --history: Generate a Jepsen log (default: false)"
	echo "  --h, --help: Display this help message"
	exit 1
}

# Parse command line options
while getopts ":c:o:-:" opt; do
	case $opt in
	c)
		num_client_processes=$OPTARG
		;;
	o)
		num_operations=$OPTARG
		;;
	-)
		case "${OPTARG}" in
		history)
			generate_history=true
			;;
		clean)
			remove_data=true
			;;
		help | h)
			usage
			;;
		*)
			echo "Invalid option: --$OPTARG" >&2
			usage
			;;
		esac
		;;
	\?)
		echo "Invalid option: -$OPTARG" >&2
		usage
		;;
	:)
		echo "Option -$OPTARG requires an argument." >&2
		usage
		;;
	esac
done

trap 'kill_servers; exit' EXIT

# Make sure executables are built.
cd cmd/kv-server
go build
cd ../kv-client
go build
cd ../..

# Start the key-value servers.
for ((i = 1; i <= 5; i++)); do
	# Bootstrap one of the servers.
	if [ $i -eq 1 ]; then
		cmd/kv-server/kv-server -id "n$i" -data "data/n$i" bootstrap -c $cluster
	fi
	cmd/kv-server/kv-server -id "n$i" -data "data/n$i" start -a "127.0.0.$i:8081" -ra "127.0.0.$i:8080" &
done

# Wait for a bit to allow servers to start up and for a leader to be elected.
sleep 2

# Start client processes.
client_pids=()
for ((i = 1; i <= num_client_processes; i++)); do
	(
		client=$i
		for ((j = 1; j <= num_operations; j++)); do
			operation=$((RANDOM % 2))
			key="x"
			value=$((RANDOM % 5 + 1))

			# Execute the client with random command and value.
			if [ $operation -eq 0 ]; then
				if $generate_history; then
					echo "{:process $client :type :invoke :f :read :value nil}" >>$jepsen_log
				fi
				result=$(cmd/kv-client/kv-client -c $kv_servers get -k $key)
				if [ -z "$result" ]; then
					result="nil"
				fi
				if [[ $result =~ $timeout_pattern && $generate_history ]]; then
					echo "{:process $client :type :fail :f :read :value timeout}" >>$jepsen_log
					exit 1
				elif $generate_history; then
					echo "{:process $client :type :ok :f :read :value $result}" >>$jepsen_log
				fi
			else
				if $generate_history; then
					echo "{:process $client :type :invoke :f :write :value $value}" >>$jepsen_log
				fi
				result=$(cmd/kv-client/kv-client -c $kv_servers put -k $key -v $value)
				if [[ $result =~ $timeout_pattern && $generate_history ]]; then
					echo "{:process $client :type :info :f :write :value timeout}" >>$jepsen_log
				elif $generate_history; then
					echo "{:process $client :type :ok :f :write :value $result}" >>$jepsen_log
				fi
			fi
		done
	) &
	client_pids+=($!)
done

# Wait for each client process to finish submitting operations.
# Kill off two of the key-value servers.
servers_killed=0
while clients_running; do
	sleep 5
	if [ $servers_killed -lt 2 ]; then
		mapfile -t server_pids < <(pgrep -f "kv-server")
		num_servers=${#server_pids[@]}
		index=$((RANDOM % num_servers))
		crash_server=${server_pids[$index]}
		kill "$crash_server"
		servers_killed=$((servers_killed + 1))
	fi
done

# Kill all of the key-value servers.
kill_servers
sleep 1

if $remove_data; then
	rm -r data
fi
