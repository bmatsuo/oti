// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// oti.go [created: Thu, 20 Mar 2014]

/*
The oti command provides a set of utilities for working with short-lived AWS
instances. For usage information pass oti the -h flag.

	oti -h
*/
package main

import (
	"github.com/bmatsuo/oti/oticonfig"
	"github.com/bmatsuo/oti/otisub"

	"flag"
	"log"
	"os"
)

var OTIVersion = "0.1"
var OTIAgent = "oti"

// default configuration informatio
var ConfigPath = "oti.json"
var Config = &oticonfig.C{
	AwsKeyPath: "aws_credentials.json",
	TagPrefix:  "co.bmats.oti.",
}

var Log = log.New(os.Stderr, "", 0)

func main() {
	fs := flag.NewFlagSet("oti", flag.ExitOnError)
	fs.StringVar(&Config.PackerDir, "p", Config.PackerDir, "packer file directory")
	fs.StringVar(&ConfigPath, "c", ConfigPath, "config file location")
	fs.Usage = func() {
		Log.Println("usage: oti [options] command")
		Log.Println()
		Log.Println("options:")
		fs.PrintDefaults()
		Log.Println()
		Log.Println("commands:")
		for _, cmd := range otisub.GetAll() {
			Log.Print("\t", cmd.Name())
		}
		Log.Println()
		Log.Println("for details about a specific command:")
		Log.Println("\toti command -h")
	}
	fs.Parse(os.Args[1:])

	cmdname, args := getargs(fs.Args(), "lifecycle")
	cmd := otisub.Get(cmdname)
	if cmd == nil {
		Log.Printf("no command %q; exiting", cmdname)
		Log.Fatal("for a list of commands run oti -h")
	}

	err := oticonfig.Read(ConfigPath, Config)
	if err != nil {
		if os.IsNotExist(err) {
			Log.Println("warning: config file not found. using defaults.")
		} else {
			Log.Fatal("error reading config: ", err)
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
