// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// instances.go [created: Sun, 23 Mar 2014]

/*

Inspect session instances

BUG this command does nothing

run a session through its full lifecycle from launch to termination.

	oti instances session-id

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
	"sync"
)

var instances = otisub.Register("instances", func(args []string) {
	fs := otisub.FlagSet(flag.ExitOnError, "instances", "session-id")
	fs.Parse(args)
	args = fs.Args()
	if len(args) != 1 {
		Log.Fatal("missing argument")
	}

	session := SessionId(args[0])
	// TODO validate session

	awsauth, err := Config.AwsAuth()
	if err != nil {
		Log.Fatalln("error reading aws credentials: ", err)
	}

	sisch := make(chan SessionInstances, len(aws.Regions))
	go DescribeSessionInstances(awsauth, session, sisch)
	for sis := range sisch {
		for _, resn := range sis.Reservations {
			for _, inst := range resn.Instances {
				cols := []string{
					sis.Region.Name,
					inst.InstanceId,
					inst.State.Name,
					inst.DNSName,
				}
				fmt.Println(strings.Join(cols, "\t"))
			}
		}
	}
})

// closes sisch on return
func DescribeSessionInstances(auth aws.Auth, session SessionId, sisch chan<- SessionInstances) {
	defer close(sisch)
	wg := new(sync.WaitGroup)
	for _, r := range Ec2Regions(false) {
		r := r
		wg.Add(1)
		go func() {
			ec2 := awsec2.New(auth, r)
			resns, err := describeSessionInstances(ec2, session)
			if err != nil {
				sisch <- SessionInstances{Region: r, SessionId: session, Err: err}
			} else {
				sisch <- SessionInstances{Region: r, SessionId: session, Reservations: resns}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func describeSessionInstances(ec2 *awsec2.EC2, session SessionId) ([]awsec2.Reservation, error) {
	filter := awsec2.NewFilter()
	if session != "" {
		filter.Add("tag:"+Config.Ec2Tag(otitag.SessionId), string(session))
	} else {
		filter.Add("tag-key", Config.Ec2Tag(otitag.SessionId))
	}

	resp, err := ec2.DescribeInstances(nil, filter)
	if err != nil {
		return nil, err
	}

	return resp.Reservations, nil
}

type SessionInstances struct {
	Region       aws.Region
	SessionId    SessionId
	Reservations []awsec2.Reservation
	Err          error
}

func NewSessionInstances(r aws.Region) *SessionInstances {
	return &SessionInstances{Region: r}
}

func (sis *SessionInstances) Append(res ...awsec2.Reservation) {
	sis.Reservations = append(sis.Reservations, res...)
}
