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

	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// the json configuration for oti
type C struct {
	// see func (c *C) AwsKey()
	AwsKeyPath string `json:",omitempty"`

	// a directory containing packer manifests.
	// see func (c *C) Packer(string)
	PackerDir string `json:",omitempty"`

	Ec2 Ec2
}

type Ec2 struct {
	// see func (c *C) Ec2Tag(otitag.OTITag)
	// oti gives this a default value.
	TagPrefix string `json:",omitempty"`

	// region specific configuration
	Regions []struct {
		// ec2 key name (recommended). overrideable per instance
		KeyName string `json:",omitempty"`

		// security groups. additional groups can be added per instance.
		// security groups with neither Id or Name are ignored.
		SecurityGroups []struct {
			Id   string `json:",omitempty"`
			Name string `json:",omitempty"`
		} `json:",omitempty"`
	}
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
func (c *C) Packer(name string) (*Packer, error) {
	var p Packer
	ppath := filepath.Join(c.PackerDir, name+".json")
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
func (c *C) Packers() ([]string, error) {
	ps, err := filepath.Glob(filepath.Join(c.PackerDir, "*.json"))
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

type Packer struct {
	Vars         *jsontree.JsonTree
	Builders     []*jsontree.JsonTree
	Provisioners []*jsontree.JsonTree
}
