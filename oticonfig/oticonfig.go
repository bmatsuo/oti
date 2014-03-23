// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// oticonfig.go [created: Thu, 20 Mar 2014]

/*
configuration for oti.
*/
package oticonfig

import (
	"github.com/bmatsuo/go-jsontree"
	"github.com/bmatsuo/oti/otitag"
	"github.com/crowdmob/goamz/aws"
	awsec2 "github.com/crowdmob/goamz/ec2"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// the json configuration for oti
type C struct {
	// packer manifest configuration
	Packer Packer `json:",omitempty"`

	// see func (c *C) AwsKey()
	AwsKeyPath string `json:",omitempty"`

	// default Ec2 deployment configurations
	Ec2 Ec2 `json:",omitempty"`

}

type Packer struct {
	// a directory containing packer manifests.
	// see func (c *C) Packer(string)
	ManifestDir string `json:",omitempty"`

	// a tag which identifies all images constructed with amazon builders.
	NameTag string `json:",omitempty"`

	// the name of a tag in packer which has the template "{{isotime}}".
	BuildDateTag string `json:",omitempty"`

	// a tag containing a sematic version number identifying it among images
	// with the same name.
	VersionTag string `json:",omitempty"` // not used
}

type Ec2 struct {
	// see func (c *C) Ec2Tag(otitag.OTITag)
	TagPrefix string `json:",omitempty"`

	// region specific configuration
	Regions []Ec2Region
}

type Ec2Region struct {
	// a unique identifier for the object. required if more than one region
	// profile is defined for the same RegionName.
	Id string `json:",omitempty"`

	// an ec2 canonical region name (e.g. "us-east-1"). required
	RegionName string

	// ec2 key name (recommended). overrideable per instance
	KeyName string `json:",omitempty"`

	// security groups. additional groups can be added per instance.
	SecurityGroups []Ec2SecurityGroup `json:",omitempty"`
}

// security groups with neither Id or Name are ignored.
type Ec2SecurityGroup struct {
	Id   string `json:",omitempty"`
	Name string `json:",omitempty"`
}

// unmarshal json data stored at path into c. any error encountered is returned.
func Read(path string, c *C) error {
	configp, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(configp, c)
	if err != nil {
		return err
	}
	return nil
}

// returns name prefixed with c.TagPrefix
func (c *C) Ec2Tag(tag otitag.OTITag) string {
	return c.Ec2.TagPrefix + string(tag)
}

// returns the first region config with RegionName equal to r.Name
func (c *C) Ec2Region(r aws.Region) *Ec2Region {
	for _, cr := range c.Ec2.Regions {
		if cr.RegionName == r.Name {
			return &cr
		}
	}
	return nil
}

// BUG uses the first region config with right name
func (c *C) Ec2KeyName(r aws.Region) string {
	cr := c.Ec2Region(r)
	if cr == nil {
		return ""
	}

	return cr.KeyName
}

// BUG uses the first region config with right name
func (c *C) Ec2SecurityGroups(r aws.Region) []awsec2.SecurityGroup {
	cr := c.Ec2Region(r)
	if cr == nil {
		return nil
	}

	n := len(cr.SecurityGroups)
	if n == 0 {
		return nil
	}

	sgs := make([]awsec2.SecurityGroup, n)
	for i, sg := range cr.SecurityGroups {
		if sg == (Ec2SecurityGroup{}) {
			continue
		}
		sgs[i].Id = sg.Id
		sgs[i].Name = sg.Name
	}

	return sgs
}

// like c.AwsKey() but returns an aws.Auth type
func (c *C) AwsAuth() (aws.Auth, error) {
	key, err := c.AwsKey()
	if err != nil {
		return aws.Auth{}, err
	}
	auth := aws.Auth{
		AccessKey: key.AccessKey,
		SecretKey: key.SecretKey,
	}
	return auth, nil
}

// unmarshal the json data stored in c.AwsKeyPath into a new AwsKey. return
// any error encountered.
func (c *C) AwsKey() (*AwsKey, error) {
	// TODO stat and warn if permissions are not strict

	keyp, err := ioutil.ReadFile(c.AwsKeyPath)
	if err != nil {
		return nil, err
	}

	var k AwsKey
	err = json.Unmarshal(keyp, &k)
	if err != nil {
		return nil, err
	}

	if k.AccessKey == "" {
		return nil, fmt.Errorf("missing AccessKey")
	}

	if k.SecretKey == "" {
		return nil, fmt.Errorf("missing SecretKey")
	}

	return &k, nil
}

// unmarshal a packer file by name. see c.Packers() for details about
// names.
func (c *C) PackerManifest(name string) (*PackerManifest, error) {
	var p PackerManifest
	ppath := filepath.Join(c.Packer.ManifestDir, name+".json")
	pp, err := ioutil.ReadFile(ppath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(pp, &p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// return the name of packer manifests in c.PackerDir. the name of the
// manifest is the file basename (without the ".json" extension).
func (c *C) PackerManifestNames() ([]string, error) {
	ps, err := filepath.Glob(filepath.Join(c.Packer.ManifestDir, "*.json"))
	if err != nil {
		return nil, err
	}
	for i := range ps {
		ps[i] = strings.TrimSuffix(filepath.Base(ps[i]), ".json")
	}
	return ps, nil
}

type AwsKey struct {
	AccessKey string
	SecretKey string
}

type PackerManifest struct {
	Vars         *jsontree.JsonTree
	Builders     []*jsontree.JsonTree
	Provisioners []*jsontree.JsonTree
}
