// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// launch.go [created: Thu, 20 Mar 2014]

/*

Launch instances

the "launch" command can be used to spin up one or more new ec2 instances.

	oti launch name [directive ...] [-- name ...]

oti-launch locates an image for each name provided and launches a specified
number of instances for each image. the instances are all tagged with a common
session identifier so they can be located later (e.g. for termination).

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
	"sort"
	"strconv"
	"strings"
	"sync"
)

var launch = otisub.Register("launch", func(args []string) {
	fs := otisub.FlagSet(flag.ExitOnError, "inspect", "imagename [directive ...] ...")
	_sessionType := fs.String("s", "", "session type for management purposes")
	_keyname := fs.String("keyname", "", "override the config KeyName for the region")
	_secgroups := fs.String("secgroup", "", "security groups to add to the instances")
	region := fs.String("r", "us-east-1", "region to run instances in")
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

	awsregion := aws.Regions[*region]
	if awsregion.Name == "" {
		Log.Fatal("unknown ec2 region %q", *region)
	}
	awsauth, err := Config.AwsAuth()
	if err != nil {
		Log.Fatalln("error reading aws credentials: ", err)
	}

	keyname := *_keyname
	if keyname == "" {
		keyname = Config.Ec2KeyName(awsregion)
	}
	secgroups := Config.Ec2SecurityGroups(awsregion)
	secgroups = append(secgroups, func() []awsec2.SecurityGroup {
		if *_secgroups == "" {
			return nil
		}
		return GuessSecurityGroups(strings.Split(*_secgroups, ","))
	}()...)

	ec2 := awsec2.New(awsauth, awsregion)

	// find images based on manifest names (if no image is explicitly specified)
	for _, mft := range ManifestsNeedingImageLookup(umfts) { // mft points into mfts
		images, err := LookupImages(ec2, mft.Name, "")
		if err != nil {
			Log.Fatal("error locating image ids: ", err)
		}

		if len(images) > 0 {
			builddatetag := Config.Images.BuildDateTag
			if builddatetag == "" {
				Log.Fatal("no build date tag to order images")
			}

			SortImages(images, builddatetag, true)
			if len(images) == 0 {
				Log.Fatal("unable to locate images")
			}
		}

		mft.Ec2ImageId = images[0].Id
	}

	for _, m := range umfts {
		if m.Name == "" {
			Log.Fatalf("manifest missing a name")
		}
		if DEBUG {
			Log.Printf("%#v", m)
		}
	}

	sessionType := *_sessionType
	if sessionType == "" {
		if len(umfts) == 1 {
			sessionType = umfts[0].Name
		} else {
			sessionType = "session"
		}
	}

	sessionId, err := NewSessionId(sessionType)
	if err != nil {
		Log.Fatal(err)
	}

	fmt.Println(sessionId) // to stdout
	if DEBUG {
		Log.Println("session id: ", sessionId)
	}

	mfts, err := BuildSystemLaunchManifests(ec2, sessionId, keyname, secgroups, umfts)
	if err != nil {
		Log.Fatalln(err)
	}

	var haserrors bool
	done := new(sync.WaitGroup)
	ich := make(chan Instances)
	_ich := make(chan []Instances, 1)
	for _, m := range mfts {
		if DEBUG {
			Log.Printf("launching manifest %#v", m)
		}
		done.Add(1)
		go func(m LaunchManifest) {
			RunInstances(ec2, m, ich)
			done.Done()
		}(m)
	}
	go func() {
		var _is []Instances
		defer func() { _ich <- _is; close(_ich) }()
		for is := range ich {
			if is.Err != nil {
				haserrors = true
				Log.Print(is.Err)
			} else {
				_is = append(_is, is)
				for _, inst := range is.Is {
					fmt.Printf("%s %s %s\n", is.M.Name, inst.InstanceId, inst.State.Name)
				}
			}
		}
	}()

	done.Wait()
	close(ich)
	iss := <-_ich
	_ = iss

	if haserrors {
		Log.Fatal()
	}

	// wait for instances to boot
	if *waitPending {
		Log.Fatal("waiting not implemented")
	}
})

type Instances struct {
	M   LaunchManifest
	Is  []awsec2.Instance
	Err error
}

func RunInstances(ec2 *awsec2.EC2, m LaunchManifest, c chan<- Instances) {
	is := Instances{M: m}
	defer func() { c <- is }()

	var userData []byte
	if m.Ec2.UserData != "" {
		userData = []byte(m.Ec2.UserData)
	}
	runopts := &awsec2.RunInstancesOptions{
		ImageId:        m.Ec2.ImageId,
		MinCount:       m.Min,
		MaxCount:       m.Max,
		KeyName:        m.Ec2.KeyName,
		InstanceType:   m.Ec2.InstanceType,
		SecurityGroups: m.Ec2.SecurityGroups,
		UserData:       userData,
	}
	resp, err := ec2.RunInstances(runopts)
	if err != nil {
		is.Err = fmt.Errorf("manifest %q: error running isntances %v", m.Name, err)
		return
	}
	is.Is = resp.Instances

	var ids []string
	for _, inst := range resp.Instances {
		ids = append(ids, inst.InstanceId)
	}

	tags := []awsec2.Tag{{Config.Ec2Tag(otitag.SessionId), string(m.SessionId)}}
	_, err = ec2.CreateTags(ids, tags)
	if err != nil {
		is.Err = fmt.Errorf("manifest %q: error tagging instances: %v", m.Name, err)
		return
	}
}

func GuessSecurityGroups(s []string) []awsec2.SecurityGroup {
	sgs := make([]awsec2.SecurityGroup, len(s))
	for i := range s {
		sgs[i] = GuessSecurityGroup(s[i])
	}
	return sgs
}

func GuessSecurityGroup(s string) awsec2.SecurityGroup {
	if strings.HasPrefix(s, "sg-") {
		return awsec2.SecurityGroup{Id: s}
	} else {
		return awsec2.SecurityGroup{Name: s}
	}
}

// create LaunchManifests from the given ULMs. the manifests are given the
// provided session id and, if the ULM does not specify a ec2 key name, the
// provided keyname as well.
func BuildSystemLaunchManifests(ec2 *awsec2.EC2, sessionId SessionId, keyname string, defaultSecgroups []awsec2.SecurityGroup, umfts []ULM) ([]LaunchManifest, error) {
	mfts := make([]LaunchManifest, len(umfts))

	// get real security groups.
	secgroups, err := LookupSecurityGroups(ec2, defaultSecgroups, umfts)
	if err != nil {
		return nil, fmt.Errorf("error locating up security groups: %v", err)
	}
	_defaultSecgroups := make([]awsec2.SecurityGroup, len(defaultSecgroups))
	for i, group := range defaultSecgroups {
		for _, info := range secgroups {
			sg := info.SecurityGroup
			if sg.Id == group.Id {
				_defaultSecgroups[i] = sg
				continue
			} else if sg.Name == group.Name {
				_defaultSecgroups[i] = sg
				continue
			}
		}
		if _defaultSecgroups[i].Id == "" {
			return nil, fmt.Errorf("unknown default security group %#v", group)
		}
	}

	// get key pairs TODO

	// build each LaunchManifest
	for i := range umfts {
		m := &mfts[i]
		um := &umfts[i]
		m.SessionId = sessionId
		m.Name = um.Name
		m.Min = um.Min
		m.Max = um.Max
		m.Ec2.InstanceType = um.Ec2InstanceType
		m.Ec2.UserData = um.Ec2UserData
		m.Ec2.ImageId = um.Ec2ImageId
		m.Ec2.KeyName = um.Ec2KeyName
		if m.Ec2.KeyName == "" {
			m.Ec2.KeyName = keyname
		}
		m.Ec2.SecurityGroups = append(m.Ec2.SecurityGroups, _defaultSecgroups...)
		for _, group := range um.Ec2SecGroups {
			found := false
			for _, info := range secgroups {
				if info.Id == group {
					m.Ec2.SecurityGroups = append(m.Ec2.SecurityGroups, info.SecurityGroup)
					found = true
				} else if info.Name == group {
					m.Ec2.SecurityGroups = append(m.Ec2.SecurityGroups, info.SecurityGroup)
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

func LookupSecurityGroups(ec2 *awsec2.EC2, secgroups []awsec2.SecurityGroup, mfts []ULM) ([]awsec2.SecurityGroupInfo, error) {
	var groups []awsec2.SecurityGroup
	groups = append(groups, secgroups...)
	for i := range mfts {
		groups = append(groups, GuessSecurityGroups(mfts[i].Ec2SecGroups)...)
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

// lookup images based on tags specified in the config file
func LookupImages(ec2 *awsec2.EC2, name, version string) ([]awsec2.Image, error) {
	nametag := Config.Images.NameTag
	if nametag == "" {
		return nil, fmt.Errorf("no name tag to identify images without explicit image ids")
	}
	if name == "" {
		return nil, fmt.Errorf("no image name given")
	}

	versiontag := Config.Images.VersionTag
	if version != "" && versiontag == "" {
		return nil, fmt.Errorf("no version tag to identify images with")
	}

	filter := awsec2.NewFilter()
	filter.Add("tag:"+nametag, name)
	if version != "" {
		filter.Add("tag"+versiontag, version)
	}

	resp, err := ec2.Images(nil, filter)
	if err != nil {
		return nil, err
	}

	images := resp.Images

	return images, nil
}

func SortImages(images []awsec2.Image, tagname string, reverse bool) {
	s := new(imgsort)
	s.images = images
	s.tagval = make([]string, len(images))

	for i := range images {
		for _, tag := range images[i].Tags {
			if tag.Key == tagname {
				s.tagval[i] = tag.Value
				break
			}
		}
	}

	if reverse {
		sort.Reverse(s)
	} else {
		sort.Sort(s)
	}
}

type imgsort struct {
	tagval []string
	images []awsec2.Image
}

func (s imgsort) Len() int           { return len(s.images) }
func (s imgsort) Less(i, j int) bool { return s.tagval[i] < s.tagval[j] }
func (s imgsort) Swap(i, j int) {
	s.tagval[i], s.tagval[j] = s.tagval[j], s.tagval[i]
	s.images[i], s.images[j] = s.images[j], s.images[i]
}

// User Launch Manifest -- information read from the command line
type ULM struct {
	Name            string   // OTI name that can be used to filter images
	LatestBuild     bool     // if no image specified use the latest built with matching tags
	Ec2UserData     string   // AWS EC2 user-data available through the instance metadata API.
	Ec2ImageId      string   // AWS EC2 image id.
	Ec2InstanceType string   // AWS EC2 instance type.
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
//	flag      alias  default     notes
//	min              1
//	max              1
//	latest           true
//	ec2type          "t1.micro"
//	ami              ""
//	keyname          ""
//	secgroup         ""
//	userdata         ""          will be base64 encoded automatically
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
		ulm.LatestBuild = true

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
			case "min", "max", "userdata", "secgroup", "ami", "keyname", "ec2type", "latest":
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
			case "latest":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					if vs[0] != "" {
						ulm.LatestBuild, err = strconv.ParseBool(vs[0])
					}
					if err == nil {
						if len(flags["ami"]) > 0 {
							err = fmt.Errorf(`cannot be specified with "ami"`)
						}
					}
				}
			case "secgroup":
				ulm.Ec2SecGroups = vs
			case "ec2type":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.Ec2InstanceType = vs[0]
				}
			case "userdata":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.Ec2UserData = vs[0]
				}
			case "ami":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else if !strings.HasPrefix(vs[0], "i-") {
					err = fmt.Errorf("invalid image id")
				} else {
					ulm.Ec2ImageId = vs[0]
				}
			case "keyname":
				if numvs > 1 {
					err = fmt.Errorf("specified multiple times")
				} else {
					ulm.Ec2KeyName = vs[0]
				}
			}
			if err != nil {
				return retErr(ulmFlagErr(k, err))
			}
		}

		if ulm.Ec2ImageId != "" {
			ulm.LatestBuild = false
		}

		if ulm.Min > ulm.Max {
			return retErr(ulmErr(fmt.Errorf(`"min" is greater than "max"`)))
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
		UserData       string                 // configured by the user
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
