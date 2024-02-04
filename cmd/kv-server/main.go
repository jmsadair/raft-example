package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmsadair/raft"
	"github.com/jmsadair/raft-example/cmd/utils"
	"github.com/jmsadair/raft-example/pkg/server"
	"github.com/jmsadair/raft/logging"
	"github.com/urfave/cli/v2"
)

var logLevelMap = map[string]logging.Level{
	"debug": logging.Debug,
	"info":  logging.Info,
	"warn":  logging.Warn,
	"error": logging.Error,
	"fatal": logging.Fatal,
}

func bootstrap(cCtx *cli.Context) error {
	id := cCtx.String("id")
	dataPath := cCtx.String("data")
	cluster := cCtx.StringSlice("cluster")
	logging := cCtx.String("log")

	level, ok := logLevelMap[logging]
	if !ok {
		return errors.New("invalid log level, must be one of: debug, info, warn, error, or fatal")
	}

	configuration, err := utils.ParseCluster(cluster)
	if err != nil {
		return err
	}

	fsm := server.NewKeyValueStore()

	node, err := raft.NewRaft(
		id,
		configuration[id],
		fsm,
		dataPath,
		raft.WithLogLevel(level),
	)
	if err != nil {
		return err
	}

	if err := node.Bootstrap(configuration); err != nil {
		return err
	}

	return nil
}

func serve(cCtx *cli.Context) error {
	id := cCtx.String("id")
	dataPath := cCtx.String("data")
	kvAddress := cCtx.String("kvAddress")
	raftAddress := cCtx.String("raftAddress")
	logging := cCtx.String("log")

	level, ok := logLevelMap[logging]
	if !ok {
		return errors.New("invalid log level, must be one of: debug, info, warn, error, or fatal")
	}

	fsm := server.NewKeyValueStore()

	node, err := raft.NewRaft(id, raftAddress, fsm, dataPath, raft.WithLogLevel(level))
	if err != nil {
		return err
	}

	server, err := server.NewServer(id, kvAddress, node)
	if err != nil {
		return err
	}

	if err := server.Start(); err != nil {
		return err
	}

	defer server.Stop()
	quitCh := make(chan os.Signal, 1)
	signal.Notify(quitCh, syscall.SIGINT, syscall.SIGTERM)
	<-quitCh

	return nil
}

func main() {
	app := &cli.App{
		Name:                 "kv-server",
		Usage:                "a demo key-value server",
		Description:          "this is a simple key-value server that is replicated using raft - it should only be used as a demonstration and for testing purposes",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "ID of this server", Required: true},
			&cli.StringFlag{
				Name:    "data",
				Aliases: []string{"d"},
				Value:   "data",
				Usage:   "directory where server data will be stored",
			},
			&cli.StringFlag{
				Name:    "log",
				Aliases: []string{"l", "logging"},
				Value:   "debug",
				Usage:   "logging level (debug, info, warn, error, fatal)",
			},
		},
		Commands: []*cli.Command{
			{
				Name:        "bootstrap",
				Description: "The 'bootstrap' command initializes raft with the specified configuration",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:     "cluster",
						Aliases:  []string{"c"},
						Usage:    "all IDs and addresses of cluster members",
						Required: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					return bootstrap(cCtx)
				},
			},
			{
				Name:        "start",
				Description: "The 'start' command starts the key-value server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "kvAddress",
						Aliases:  []string{"a"},
						Usage:    "listen address of key-value server",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "raftAddress",
						Aliases:  []string{"ra"},
						Usage:    "listen address of raft server",
						Required: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					return serve(cCtx)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
