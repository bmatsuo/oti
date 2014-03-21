// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// terminate.go [created: Thu, 20 Mar 2014]

package main

import (
	"github.com/bmatsuo/oti/otisub"

	"flag"
	"fmt"
)

var terminate = otisub.Register("terminate", func(args []string) {
	flag.Usage = otisub.Usage("terminate")
	flag.Parse()

	fmt.Println("Kill! Kill! KILLL!!!")
})
