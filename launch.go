// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// launch.go [created: Thu, 20 Mar 2014]

/*
Launch instances

the "launch" command can be used to spin up one or more new ec2 instances.

	oti launch -h

*/
package main

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/bmatsuo/oti/otisub"
	"github.com/bmatsuo/oti/otitag"
	"github.com/crowdmob/goamz/aws"
	awsec2 "github.com/crowdmob/goamz/ec2"

	"flag"
	"fmt"
	"strconv"
	"strings"
)

var launch = otisub.Register("launch", func(args []string) {
	fs := otisub.FlagSet(flag.ExitOnError, "inspect", "imagename [directive ...] ...")
	//keypath := fs.String("i", "", "ssh key path")
	sessionType := fs.String("s", "launch", "session type for management purposes")
	waitPending := fs.Bool("w", false, "wait while instances are 'pending'")
	fs.Parse(args)
	args = fs.Args()

	umfts, err := ParseUserLaunchManifest(args)
	if err != nil {
		Log.Fatal(err)
	}

	if len(umfts) == 0 {
		Log.Fatal("no manifests")
	}

	awsregion := aws.USEast
	awsauth, err := Config.AwsAuth()
	if err != nil {
		Log.Fatalln("error reading aws credentials: ", err)
	}

	ec2 := awsec2.New(awsauth, awsregion)

	for _, mft := range ManifestsNeedingImageLookup(umfts) { // mft points into mfts
		images, err := LookupImages(ec2, mft.Name)
		if err != nil {
			Log.Fatal("error locating image ids: ", err)
		}
		if len(images) > 0 {
			Log.Fatal("ambigous results: %v", images)
		}
		mft.Ec2ImageId = images[0].Id
	}

	for _, m := range umfts {
		Log.Printf("%#v", m)
	}

	sessionId, err := NewSessionId(*sessionType)
	if err != nil {
		Log.Fatal(err)
	}
	Log.Println("oti session id: ", sessionId)

	// TODO find images based on manifest name

	mfts, err := BuildSystemLaunchManifests(ec2, sessionId, umfts)

	for _, m := range mfts {
		Log.Printf("%#v", m)
	}

	var haserrors bool
	for _, m := range mfts {
		runopts := &awsec2.RunInstancesOptions{
			ImageId:        m.Ec2.ImageId,
			MinCount:       m.Min,
			MaxCount:       m.Max,
			KeyName:        m.Ec2.KeyName,
			InstanceType:   m.Ec2.InstanceType,
			SecurityGroups: m.Ec2.SecurityGroups,
		}
		resp, err := ec2.RunInstances(runopts)
		if err != nil {
			haserrors = true
			Log.Printf("error running %q: %v", m.Name, err)
			continue
		}

		var instanceIds []string
		for _, inst := range resp.Instances {
			instanceIds = append(instanceIds, inst.InstanceId)
		}

		Log.Printf("started instances: %v", instanceIds)

		_, err = ec2.CreateTags(instanceIds, []awsec2.Tag{
			{Config.Ec2Tag(otitag.SessionId), string(m.SessionId)},
		})
		if err != nil {
			haserrors = true
			Log.Printf("unable to tag instances: %v", err)
		}
	}
	if haserrors {
		Log.Fatal()
	}

	// wait for instances to boot
	if *waitPending {
		Log.Fatal("waiting not implemented")
	}

	fmt.Println("LAUNCH!")
})

func BuildSystemLaunchManifests(ec2 *awsec2.EC2, sessionId SessionId, umfts []ULM) ([]LaunchManifest, error) {
	mfts := make([]LaunchManifest, len(umfts))
	secgroups, err := LookupSecurityGroups(ec2, umfts)
	if err != nil {
		return nil, fmt.Errorf("error locating up security groups: %v", err)
	}

	for i := range umfts {
		mfts[i].SessionId = sessionId
		mfts[i].Name = umfts[i].Name
		mfts[i].Min = umfts[i].Min
		mfts[i].Max = umfts[i].Max
		mfts[i].Ec2.InstanceType = umfts[i].Ec2InstanceType
		mfts[i].Ec2.ImageId = umfts[i].Ec2ImageId
		mfts[i].Ec2.KeyName = umfts[i].Ec2KeyName
		for _, group := range umfts[i].Ec2SecGroups {
			found := false
			for _, info := range secgroups {
				if info.Id == group {
					mfts[i].Ec2.SecurityGroups = append(mfts[i].Ec2.SecurityGroups, info.SecurityGroup)
					found = true
				} else if info.Name == group {
					mfts[i].Ec2.SecurityGroups = append(mfts[i].Ec2.SecurityGroups, info.SecurityGroup)
					found = true
				}
			}
			if !found {
				return nil, fmt.Errorf("unknown security group: %q", group)
			}
		}
	}

	return mfts, nil
}

func LookupSecurityGroups(ec2 *awsec2.EC2, mfts []ULM) ([]awsec2.SecurityGroupInfo, error) {
	var groups []awsec2.SecurityGroup
	for i := range mfts {
		for _, group := range mfts[i].Ec2SecGroups {
			if strings.HasPrefix(group, "sg-") {
				groups = append(groups, awsec2.SecurityGroup{Id: group})
			} else {
				groups = append(groups, awsec2.SecurityGroup{Name: group})
			}
		}
	}

	resp, err := ec2.SecurityGroups(groups, nil)
	if err != nil {
		return nil, err
	}

	return resp.Groups, nil
}

func ManifestsNeedingImageLookup(ulms []ULM) []*ULM {
	var _ulms []*ULM
	for i := range ulms {
		if ulms[i].Ec2ImageId == "" {
			_ulms = append(_ulms, &ulms[i])
		}
	}
	return _ulms
}

// use the packer config to locate images on ec2.
func LookupImages(ec2 *awsec2.EC2, name string) ([]awsec2.Image, error) {
	return nil, fmt.Errorf("unimplemented")
}

// User Launch Manifest -- information read from the command line
type ULM struct {
	Name            string   // OTI name that can be used to filter images
	Ec2ImageId      string   // AWS EC2 image id.
	Ec2InstanceType string   // AWS EC2 instance type.
	IdentityFile    string   // SSH .pem file
	Ec2KeyName      string   // AWS EC2 key pair name
	Ec2SecGroups    []string // Security groups to assign the instances
	Min, Max        int      // may not be empty
}

type ArgumentError struct {
	i   int
	err error
}

func (err ArgumentError) Error() string {
	return fmt.Sprintf("argument %d: %v", err.i, err.err)
}

var ErrEndOfArgs = ArgumentError{-1, fmt.Errorf("no more arguments")}

// parses a set launch manifest. manifests have the form
//	name [ flag[=val] ... ] -- ...
// for reference use the following list of flags and the default values
//	flag      alias  default
//	min              1
//	max              1
//	ec2type          "t1.micro"
//	ami              ""
//	identity  i      ""
//	keyname          ""
//	secgroup         ""
func ParseUserLaunchManifest(args []string) ([]ULM, error) {
	ulms := make([]ULM, 0, len(args))
	sepseq := "--"

	parseUlm := func(args []string) (ulm ULM, rest []string, err error) {
		rest = args

		if len(rest) == 0 {
			return ULM{}, nil, ErrEndOfArgs
		}

		if rest[0] == sepseq {
			err := fmt.Errorf("unexpected separator sequence %v", sepseq)
			return ULM{}, nil, err
		}

		ulm.Name, rest = args[0], rest[1:]

		// set defaults
		ulm.Min, ulm.Max = 1, 1
		ulm.Ec2InstanceType = "t1.micro"

		retErr := func(err error) (ULM, []string, error) {
			return ULM{}, nil, err
		}
		ulmErr := func(err error) error { return fmt.Errorf("%v %v", ulm.Name, err) }
		ulmFlagErr := func(key string, err error) error {
			return ulmErr(fmt.Errorf("invalid flag %q: %v", key, err))
		}

		flags := make(map[string][]string)
		for len(rest) > 0 && rest[0] != sepseq {
			var head string
			head, rest = rest[0], rest[1:]
			key, value := head, ""

			pair := strings.SplitN(key, "=", 2)
			if len(pair) == 2 {
				key, value = pair[0], pair[1]
			}

			switch key {
			case "min", "max", "secgroup", "ami", "keyname", "ec2type":
			case "identity", "i":
				key = "identity" // alias
			default:
				err := fmt.Errorf("unexpected flag %v", key)
				return retErr(ulmErr(err))
			}

			flags[key] = append(flags[key], value)
		}

		for k, vs := range flags {
			var err error
			numvs := len(vs)
			switch k {
			case "min":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.Min, err = strconv.Atoi(vs[0])
				}
			case "max":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.Max, err = strconv.Atoi(vs[0])
				}
			case "secgroup":
				ulm.Ec2SecGroups = vs
			case "ec2type":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.Ec2InstanceType = vs[0]
				}
			case "ami":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.Ec2ImageId = vs[0]
				}
			case "keyname":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.Ec2KeyName = vs[0]
				}
			case "identity":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.IdentityFile = vs[0]
				}
			}
			if err != nil {
				return retErr(ulmFlagErr(k, err))
			}
		}

		if len(ulm.Ec2SecGroups) == 0 {
			return retErr(ulmFlagErr("secgroup", fmt.Errorf("required for now")))
		}

		if ulm.Ec2ImageId == "" {
			return retErr(ulmFlagErr("ami", fmt.Errorf("required for now")))
		}

		if ulm.IdentityFile == "" {
			return retErr(ulmFlagErr("identity", fmt.Errorf("required for now")))
		}

		if ulm.Ec2KeyName == "" {
			return retErr(ulmFlagErr("keyname", fmt.Errorf("required for now")))
		}

		if ulm.Min > ulm.Max {
			return retErr(ulmErr(err))
		}

		if len(rest) > 0 && rest[0] == sepseq {
			return ulm, rest[1:], nil
		} else {
			return ulm, nil, nil
		}
	}

	for len(args) > 0 {
		var ulm ULM
		var err error
		ulm, args, err = parseUlm(args)
		if err != nil {
			return nil, err
		}
		ulms = append(ulms, ulm)
	}

	return ulms, nil
}

type LaunchManifest struct {
	Name      string    // configured by the user
	Min, Max  int       // configured by the user
	SessionId SessionId // generated at runtime
	Ec2       struct {
		ImageId        string                 // located AWS image id
		InstanceType   string                 // configured by the user
		KeyName        string                 // configured by the user or generated at run-time
		SecurityGroups []awsec2.SecurityGroup // configured by the user or created at runtime
	}
}

type SessionId string

func NewSessionId(sessiontype string) (SessionId, error) {
	if strings.Contains(sessiontype, ":") {
		return "", fmt.Errorf("session type cannot contain ':'")
	}
	if sessiontype == "" {
		sessiontype = "session"
	}
	sid := SessionId(fmt.Sprintf("%s:%v", sessiontype, uuid.New()))
	return sid, nil
}

func (sid SessionId) Type() string {
	return strings.SplitN(string(sid), ":", 2)[0]
}
