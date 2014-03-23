// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// terminate.go [created: Thu, 20 Mar 2014]

/*

Terminate instances

the "terminate" command can be used to terminate oti sessions.

	oti terminate -s session-type
	oti terminate session-id ...

when -s is given oti terminates all sessions of a given type.  when session
id(s) are specified oti terminates all instances belonging to the session(s).
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
	"fmt"
	"strings"
)

var terminate = otisub.Register("terminate", func(args []string) {
	opts := new(TerminateOptions)
	fs := otisub.FlagSet(flag.ExitOnError, "terminate", "session-id ...")
	exceptstates := fs.String("except-states", "shutting-down,terminated", "do not try to terminate these instances")
	onlystates := fs.String("only-states", "*", "terminate only instances in one of these states")
	fs.StringVar(&opts.SessionType, "s", "", "terminate all sessions with this type")
	region := fs.String("r", "us-east-1", "ec2 region to look for instances")
	fs.BoolVar(&opts.WaitShuttingDown, "w", false, "wait while instances are 'shutting-down'")
	fs.Parse(args)
	args = fs.Args()

	opts.ExceptStates = strings.Split(*exceptstates, ",")
	opts.OnlyStates = strings.Split(*onlystates, ",")

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
	ExceptStates     []string
	OnlyStates       []string
	SessionType      string
	WaitShuttingDown bool
}

// takes a list of target identifiers to terminate and options.
func TerminateMain(targets []string, opts *TerminateOptions) {
	if len(targets) == 0 && opts.SessionType == "" {
		Log.Println("no targets...")
		return
	}

	ec2 := awsec2.New(opts.Auth, opts.Region)

	resvns, err := LocateTargetInstances(ec2, targets, opts.SessionType, opts.OnlyStates, opts.ExceptStates)
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

	if len(instanceIds) == 0 {
		Log.Println("no matching instances. run `oti sessions` to inspect sessions")
		return
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
func LocateTargetInstances(ec2 *awsec2.EC2, targets []string, sessiontype string, onlystates, exceptstates []string) ([]awsec2.Reservation, error) {
	if ec2 == nil {
		panic("nil ec2 connection")
	}

	if len(targets) == 0 && sessiontype == "" {
		return nil, fmt.Errorf("no target tessions")
	}

	sessionidtag := Config.Ec2Tag(otitag.SessionId)

	filter := awsec2.NewFilter()
	for i := range targets {
		filter.Add("tag:"+sessionidtag, targets[i])
	}
	if sessiontype != "" {
		filter.Add("tag-key", sessionidtag)
	}

	resp, err := ec2.DescribeInstances(nil, filter)
	if err != nil {
		return nil, err
	}

	rs := resp.Reservations

	matchesState := func(states []string) func(*awsec2.Instance) bool {
		return func(inst *awsec2.Instance) bool {
			for _, state := range states {
				if state == "*" {
					return true
				}
				if state == inst.State.Name {
					return true
				}
			}
			return false
		}
	}

	for i := range rs {
		resp.Reservations[i].Instances = FilterInstances(FilterInstances(FilterInstances(
			resp.Reservations[i].Instances,
			func(inst *awsec2.Instance) bool {
				for _, tag := range inst.Tags {
					if tag.Key == sessionidtag {
						if SessionId(tag.Value).Type() == sessiontype {
							return true
						}
						if DEBUG {
							Log.Printf("discarding instance %q with tag %v",
								inst.InstanceId, tag)
						}
						return false
					}
				}
				return false
			}),
			matchesState(onlystates)),
			func(inst *awsec2.Instance) bool {
				return !matchesState(exceptstates)(inst)
			})
	}

	rs = FilterReservations(rs,
		func(r *awsec2.Reservation) bool { return len(r.Instances) == 0 })

	return resp.Reservations, nil
}

func FilterReservations(rs []awsec2.Reservation, fn func(*awsec2.Reservation) bool) []awsec2.Reservation {
	_rs := make([]awsec2.Reservation, 0, len(rs))
	for i := range rs {
		if fn(&rs[i]) {
			_rs = append(_rs, rs[i])
		}
	}
	return _rs
}

func FilterInstances(is []awsec2.Instance, fn func(*awsec2.Instance) bool) []awsec2.Instance {
	_is := make([]awsec2.Instance, 0, len(is))
	for i := range is {
		if fn(&is[i]) {
			_is = append(_is, is[i])
		}
	}
	return _is
}
