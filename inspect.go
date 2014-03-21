// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// inspect.go [created: Thu, 20 Mar 2014]

package main

import (
	"github.com/bmatsuo/oti/otisub"
	_ "github.com/crowdmob/goamz/aws"
	_ "github.com/crowdmob/goamz/ec2"

	"flag"
	"fmt"
)

var inspect = otisub.Register("inspect", func(args []string) {
	flag.Usage = otisub.Usage("inspect")
	flag.Parse()

	awskey, err := Config.AwsKey()
	if err != nil {
		Log.Fatal("error reading aws key: %v", err)
	}
	_ = awskey

	fmt.Println("what do we have here???")
})
