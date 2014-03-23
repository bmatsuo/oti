// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// inspect.go [created: Thu, 20 Mar 2014]

/*

Inspect sessions

the "sessions" command provides information about known oti sessions and their
resources. resources are identified by tag values.

	oti sessions -h

locates existing sessions. sessions exists merely by having instances tagged
with their session id.

BUG sessions cannot span regions

*/
package main

import (
	"github.com/bmatsuo/oti/otisub"
	"github.com/bmatsuo/oti/otitag"
	"github.com/crowdmob/goamz/aws"
	awsec2 "github.com/crowdmob/goamz/ec2"

	"flag"
	"fmt"
	"sync"
)

var sessions = otisub.Register("sessions", func(args []string) {
	fs := otisub.FlagSet(flag.ExitOnError, "sessions", "[sessionid ...]")
	fs.Parse(args)

	sessionids := fs.Args()

	awsauth, err := Config.AwsAuth()
	if err != nil {
		Log.Fatalf("error reading aws credentials: %v", err)
	}

	wg := new(sync.WaitGroup)
	for _, r := range aws.Regions {
		r := r
		if r.Name == "us-gov-west-1" {
			// shhh
			continue
		}
		if r.EC2Endpoint != "" {
			wg.Add(1)
			go func() {
				SessionsMain(awsauth, r, sessionids)
				wg.Done()
			}()
		}
	}
	wg.Wait()
})

// locate and inspect sessions, active or terminated
func SessionsMain(auth aws.Auth, region aws.Region, sessionids []string) {
	ec2 := awsec2.New(auth, region)
	sessions, err := LocateSessions(ec2, sessionids)
	if err != nil {
		Log.Fatalln("error locating instances: ", err)
	}

	// print session details to stdout
	for _, s := range sessions {
		fmt.Printf("%s\t%s\t%s\n", region.Name, s.Id, DescribeSessionInstanceStates(s))
	}
}

type Session struct {
	Id        SessionId
	Instances []awsec2.Instance
}

func LocateSessions(ec2 *awsec2.EC2, sessions []string) ([]Session, error) {
	sessionIdTag := Config.Ec2Tag(otitag.SessionId)
	filter := awsec2.NewFilter()
	if len(sessions) > 0 {
		filter.Add("tag:"+sessionIdTag, sessions...)
	} else {
		filter.Add("tag-key", Config.Ec2Tag(otitag.SessionId))
	}
	resp, err := ec2.DescribeInstances(nil, filter)
	if err != nil {
		return nil, err
	}

	simap := make(map[SessionId][]awsec2.Instance)
	for _, rsvn := range resp.Reservations {
		for _, inst := range rsvn.Instances {
			for _, tag := range inst.Tags {
				if tag.Key == sessionIdTag {
					sessionId := SessionId(tag.Value)
					simap[sessionId] = append(simap[sessionId], inst)
					break
				}
			}
		}
	}

	var ss []Session
	for id, is := range simap {
		ss = append(ss, Session{id, is})
	}

	return ss, nil
}

func DescribeSessionInstanceStates(s Session) string {
	counts := make(map[string]int, 5)
	for _, inst := range s.Instances {
		counts[inst.State.Name]++
	}
	return fmt.Sprintf("%d/%d/%d/%d/%d",
		counts["pending"], counts["running"],
		counts["shutting-down"], counts["stopped"],
		counts["terminated"])
}
