// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// lifecycle.go [created: Thu, 20 Mar 2014]

package main

import (
	"github.com/bmatsuo/oti/otisub"

	"flag"
	"fmt"
)

var lifecycle = otisub.Register("lifecycle", func(args []string) {
	flag.Usage = otisub.Usage("lifecycle")
	flag.Parse()

	fmt.Println("The circle of life.")
})
