// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// oti.go [created: Thu, 20 Mar 2014]

package main

import (
	"github.com/bmatsuo/oti/otisub"

	"flag"
	"log"
	"os"
)

// default configuration informatio
var ConfigPath = "oti.json"
var Config = &OTIConfig{
	AwsKeyPath:  "aws_credentials.json",
	ResourceTag: OTITag{"ManagingAgent", "oti"},
}

func main() {
	logger := log.New(os.Stderr, "", 0)

	fs := flag.NewFlagSet("oti", flag.ExitOnError)
	fs.StringVar(&ConfigPath, "c", ConfigPath, "config file location")
	fs.Usage = func() {
		logger.Println("usage: oti [options] command")
		logger.Println()
		logger.Println("options:")
		fs.PrintDefaults()
		logger.Println()
		logger.Println("commands:")
		for _, cmd := range otisub.GetAll() {
			logger.Print("\t", cmd.Name())
		}
		logger.Println()
		logger.Println("for details about a specific command:")
		logger.Println("\toti command -h")
	}
	fs.Parse(os.Args[1:])

	cmdname, args := getargs(fs.Args(), "lifecycle")
	cmd := otisub.Get(cmdname)
	if cmd == nil {
		logger.Printf("no command %q; exiting", cmdname)
		logger.Fatal("for a list of commands run oti -h")
	}

	err := readConfig(ConfigPath, Config)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Println("warning: config file not found. using defaults.")
		} else {
			logger.Fatal("error reading config: ", err)
		}
	}

	cmd.Main(args)
}

func getargs(args []string, defcmd string, defargs ...string) (subcmd string, subargs []string) {
	if len(args) == 0 {
		return defcmd, defargs
	}
	return args[0], args[1:]
}
