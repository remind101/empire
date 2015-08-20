package main

import (
	"os"

	"github.com/codegangsta/cli"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	app := cli.NewApp()
	app.Name = "kinesumer"
	app.Usage = "A tool working with AWS Kinesis and kinesumer"
	app.Version = "0.0.0"
	app.Authors = []cli.Author{
		{
			Name:  "Tony Zou",
			Email: "<tony@tonyzou.com>",
		},
	}
	app.Commands = []cli.Command{
		cmdStatus,
		cmdTail,
	}
	app.Run(os.Args)
}
