// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// oti.go [created: Thu, 20 Mar 2014]

package main

import (
	"github.com/bmatsuo/oti/otisub"

	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()
	cmdname, args := getargs("lifecycle")
	cmd := otisub.Get(cmdname)
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "no such command: %q\n", cmdname)
	}
	cmd.Main(args)
}

func getargs(defcmd string, defargs ...string) (subcmd string, subargs []string) {
	args := flag.Args()
	if len(args) == 0 {
		return defcmd, defargs
	}
	return args[0], args[1:]
}
