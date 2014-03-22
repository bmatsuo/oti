// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// terminate.go [created: Thu, 20 Mar 2014]

/*
Terminate instances

the "terminate" command can be used to terminate one or more ec2 instances.

	oti terminate -h

*/
package main

import (
	"github.com/bmatsuo/oti/otisub"

	"flag"
	"fmt"
)

var terminate = otisub.Register("terminate", func(args []string) {
	fs := otisub.FlagSet(flag.ExitOnError, "terminate", "target ...")
	wait := fs.Bool("w", false, "wait while instances are 'shutting-down'")
	fs.Parse(args)
	args = fs.Args()

	fmt.Println("Kill! Kill! KILLL!!!")

	_ = wait
})
