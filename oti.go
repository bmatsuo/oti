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

func main() {
	logger := log.New(os.Stderr, "", 0)
	opts := new(struct {
		configpath string
	})
	fs := flag.NewFlagSet("oti", flag.ExitOnError)
	fs.StringVar(&opts.configpath, "c", "oti.json", "config file location")
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
	cmd.Main(args)
}

func getargs(args []string, defcmd string, defargs ...string) (subcmd string, subargs []string) {
	if len(args) == 0 {
		return defcmd, defargs
	}
	return args[0], args[1:]
}
