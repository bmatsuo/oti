// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// terminate.go [created: Thu, 20 Mar 2014]

/*

Terminate instances

the "terminate" command can be used to terminate oti sessions.

	oti terminate session-id ...

oti-terminate locates all instances belonging to a session and terminates them.
if all instances in the given sessions enter the 'shutting-down' state, the
command will exit with a zero exit status.

*/
package main

import (
	"github.com/bmatsuo/oti/otisub"
	"github.com/bmatsuo/oti/otitag"
	"github.com/crowdmob/goamz/aws"
	awsec2 "github.com/crowdmob/goamz/ec2"

	"flag"
)

var terminate = otisub.Register("terminate", func(args []string) {
	opts := new(TerminateOptions)
	fs := otisub.FlagSet(flag.ExitOnError, "terminate", "session-id ...")
	region := fs.String("r", "us-east-1", "ec2 region to look for instances")
	fs.BoolVar(&opts.WaitShuttingDown, "w", false, "wait while instances are 'shutting-down'")
	fs.Parse(args)
	args = fs.Args()

	auth, err := Config.AwsAuth()
	if err != nil {
		Log.Fatal("error reading aws credentials: ", err)
	}
	opts.Auth = auth

	opts.Region = aws.Regions[*region]
	if opts.Region.Name == "" {
		Log.Fatal("unknown ec2 region %q", *region)
	}

	TerminateMain(args, opts)
})

type TerminateOptions struct {
	Region           aws.Region
	Auth             aws.Auth
	WaitShuttingDown bool
}

// takes a list of target identifiers to terminate and options.
func TerminateMain(targets []string, opts *TerminateOptions) {
	if len(targets) == 0 {
		Log.Println("no targets...")
		return
	}

	ec2 := awsec2.New(opts.Auth, opts.Region)

	resvns, err := LocateTargetInstances(ec2, targets)
	if err != nil {
		Log.Fatal(err)
	}

	if len(resvns) == 0 {
		Log.Fatal("no instances found")
	}

	instanceIds := make([]string, 0)
	for _, resvn := range resvns {
		for _, inst := range resvn.Instances {
			instanceIds = append(instanceIds, inst.InstanceId)
		}
	}

	if DEBUG {
		Log.Printf("terminating instances %v", instanceIds)
	}

	resp, err := ec2.TerminateInstances(instanceIds)
	if err != nil {
		Log.Fatal(err)
	}

	for _, change := range resp.StateChanges {
		Log.Printf("%s %s (was %s)",
			change.InstanceId,
			change.CurrentState.Name,
			change.PreviousState.Name)
	}
}

// find instances tagged with target session ids
func LocateTargetInstances(ec2 *awsec2.EC2, targets []string) ([]awsec2.Reservation, error) {
	if ec2 == nil {
		panic("nil ec2 connection")
	}

	if len(targets) == 0 {
		return nil, nil
	}

	filter := awsec2.NewFilter()
	for i := range targets {
		filter.Add("tag:"+Config.Ec2Tag(otitag.SessionId), targets[i])
	}

	resp, err := ec2.DescribeInstances(nil, filter)
	if err != nil {
		return nil, err
	}

	return resp.Reservations, nil
}
