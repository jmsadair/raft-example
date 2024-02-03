package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jmsadair/raft-example/cmd/utils"
	"github.com/jmsadair/raft-example/pkg/client"
	"github.com/urfave/cli/v2"
)

func operation(cCtx *cli.Context, isPut bool) error {
	cluster := cCtx.StringSlice("cluster")
	key := cCtx.String("key")
	value := cCtx.String("value")
	timeout := cCtx.Uint("timeout")

	configuration, err := utils.ParseCluster(cluster)
	if err != nil {
		return err
	}

	client, err := client.NewClient(configuration)
	if err != nil {
		return err
	}

	if isPut {
		if _, err := client.Put(key, value, time.Duration(timeout)*time.Second); err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, value)
	} else {
		result, err := client.Get(key, time.Duration(timeout)*time.Second)
		if err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, result)
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:                 "key-value-client",
		Usage:                "a demo client for the key-value server",
		Description:          "this is a simple client for the key-value server - it should only be used as a demonstration and for testing purposes",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "cluster",
				Aliases:  []string{"c"},
				Usage:    "IDs and addresses of all key-value servers",
				Required: true,
			},
			&cli.UintFlag{
				Name:    "timeout",
				Aliases: []string{"t"},
				Usage:   "the timeout (in seconds) for the put operation",
				Value:   3,
			},
		},
		Commands: []*cli.Command{
			{
				Name:        "put",
				Description: "the 'put' command sets the provided key to the provided value",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "key",
						Aliases:  []string{"k"},
						Usage:    "the key in the put operation",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "value",
						Aliases:  []string{"v"},
						Usage:    "the value in the put operation",
						Required: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					return operation(cCtx, true)
				},
			},
			{
				Name:        "get",
				Description: "the 'get' command retrieves the value associated with the provided key",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "key",
						Aliases:  []string{"k"},
						Usage:    "the key in the get operation",
						Required: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					return operation(cCtx, false)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
