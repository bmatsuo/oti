// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// lifecycle.go [created: Thu, 20 Mar 2014]

/*
Instance lifecycle

run a session through its full lifecycle from launch to termination.

	oti lifecycle name [directive ...] [-- name ... ]

BUG this command does nothing
*/
package main

import (
	"github.com/bmatsuo/oti/otisub"

	"flag"
	"fmt"
)

var lifecycle = otisub.Register("lifecycle", func(args []string) {
	fs := otisub.FlagSet(flag.ExitOnError, "lifecycle", "imagename [directive ...] ...")
	_ = fs.Bool("w", false, "wait while instances are 'shutting-down'")
	fs.Parse(args)
	args = fs.Args()

	fmt.Println("the circle of life.")
})
