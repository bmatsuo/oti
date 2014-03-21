// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// launch.go [created: Thu, 20 Mar 2014]

package main

import (
	"github.com/bmatsuo/oti/otisub"

	"flag"
	"fmt"
)

var launch = otisub.Register("launch", func(args []string) {
	flag.Usage = otisub.Usage("launch")
	flag.Parse()

	fmt.Println("LAUNCH!")
})
