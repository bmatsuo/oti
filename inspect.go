// Copyright 2014, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// inspect.go [created: Thu, 20 Mar 2014]

/*
Inspect images and instances

the "inspect" command provides information on available images and instances
running those images.

	oti inspect -h

*/
package main

import (
	"github.com/bmatsuo/oti/otisub"
	"github.com/crowdmob/goamz/aws"
	awsec2 "github.com/crowdmob/goamz/ec2"

	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

var inspect = otisub.Register("inspect", func(args []string) {
	fs := otisub.FlagSet(flag.ExitOnError, "inspect", "[imagename ...]")
	regionName := fs.String("r", "", "an ec2 region (e.g. \"us-east-1\")")
	imageIds := fs.String("i", "", "comma separated image ids")
	fs.Parse(args)
	args = fs.Args()

	if len(args) > 0 && *imageIds != "" {
		Log.Fatal("arguments cannot be given if -i is supplied")
	}

	awskey, err := Config.AwsKey()
	if err != nil {
		Log.Fatalf("error reading aws key: %v", err)
	}
	awsauth := aws.Auth{
		AccessKey: awskey.AccessKey,
		SecretKey: awskey.SecretKey,
	}

	var imagech <-chan awsec2.Image
	if *imageIds != "" {
		if *regionName == "" {
			*regionName = "us-east-1"
			Log.Printf("unspecified region; defaulting to %q", *regionName)
		}
		if *regionName != "us-east-1" {
			panic("unsupported region") // lazy
		}
		imagech = InspectRegionImageIds(
			awsauth, aws.USEast,
			strings.Split(*imageIds, ","))
	} else {
		var names []string
		names = args
		if len(names) == 0 {
			pkrs, err := Config.PackerManifestNames()
			if err != nil {
				Log.Fatalf("error locating packer files: %v", err)
			}
			names = pkrs
		}
		if len(names) == 0 {
			Log.Fatal("no images.. sadface")
		}
		imagech = InspectRegionImageNames(awsauth, aws.USEast, names)
	}

	DescribeImages(true, imagech)
})

// inspect images specified by their amazon ids.
func InspectRegionImageIds(auth aws.Auth, region aws.Region, imageids []string) <-chan awsec2.Image {
	imagech := make(chan awsec2.Image)
	go func() {
		ec2 := awsec2.New(auth, region)

		_imagech := make(chan Image, len(imageids))
		InspectImages(ec2, imageids, nil, _imagech)
		images, err := CollectImages(_imagech)
		if err != nil {
			Log.Fatal(err)
		}
		SortImages("ServiceName", "BuildDate", images) // FIXME arbitrary choices

		defer close(imagech)
		for _, image := range images {
			imagech <- image
		}
	}()
	return imagech
}

// inspect images specified by their packer names.
func InspectRegionImageNames(auth aws.Auth, region aws.Region, names []string) <-chan awsec2.Image {
	imagech := make(chan awsec2.Image)
	go func() {
		defer close(imagech)
		ec2 := awsec2.New(auth, region)

		_imagech := make(chan Image, 10)
		filter := awsec2.NewFilter()
		filter.Add("tag:ServiceName", names...)
		go InspectImages(ec2, nil, filter, _imagech)
		images, err := CollectImages(_imagech)
		if err != nil {
			Log.Fatal(err)
		}
		SortImages("ServiceName", "BuildDate", images) // FIXME arbitrary choices
		for _, image := range images {
			imagech <- image
		}
	}()
	return imagech
}

// BUG buffers all output without flushing
func DescribeImages(header bool, images <-chan awsec2.Image) {
	w := tabwriter.NewWriter(os.Stdout, 10, 4, 2, ' ', 0)
	defer w.Flush()

	if header {
		fmt.Fprintf(w, "id\tservice\tbuild\tP/R/D/S/T\n")
	}

	for image := range images {
		pending, running, shuttingDown, stopped, terminated := 0, 0, 0, 0, 0
		fmt.Fprintf(w, "%v\t%v\t%v\t%d/%d/%d/%d/%d\n",
			image.Id,
			getImageTag(image, "ServiceName"),
			getImageTag(image, "BuildDate"),
			pending, running, shuttingDown, stopped, terminated,
		)
	}
}

func InspectImages(ec2 *awsec2.EC2, imageids []string, filter *awsec2.Filter, images chan<- Image) {
	defer close(images)
	resp, err := ec2.Images(imageids, filter)
	if err != nil {
		images <- Image{err: err}
		return
	}
	for _, image := range resp.Images {
		images <- Image{Image: image}
	}
}

type ImageErrors []error

func (err ImageErrors) Error() string {
	return fmt.Sprintf("error(s) retrieving images: %v", []error(err))
}

func CollectImages(images <-chan Image) ([]awsec2.Image, error) {
	var errs ImageErrors
	_images := make([]awsec2.Image, 0, len(images))
	for image := range images {
		if image.err != nil {
			errs = append(errs, image.err)
		} else {
			_images = append(_images, image.Image)
		}
	}
	if len(errs) == 0 {
		return _images, nil
	}
	return _images, errs
}

func SortImages(typeTag, timeTag string, images []awsec2.Image) {
	sort.Sort(isort{typeTag, timeTag, images})
}

type isort struct {
	TypeTag string
	TimeTag string
	is      []awsec2.Image
}

func (is isort) Len() int      { return len(is.is) }
func (is isort) Swap(i, j int) { is.is[i], is.is[j] = is.is[j], is.is[i] }

// BUG order is not configurable and time is descending by default
func (is isort) Less(i, j int) bool {
	if is.TypeTag != "" {
		type1 := getImageTag(is.is[i], is.TypeTag)
		type2 := getImageTag(is.is[j], is.TypeTag)
		if type1 != type2 {
			return type1 < type2
		}
	}

	if is.TimeTag != "" {
		stamp1 := getImageTag(is.is[i], is.TimeTag)
		stamp2 := getImageTag(is.is[j], is.TimeTag)
		if stamp1 != stamp2 {
			// descending order by default
			return stamp1 > stamp2
		}
	}

	return false
}

func getImageTag(image awsec2.Image, key string) string {
	for _, tag := range image.Tags {
		if tag.Key == key {
			return tag.Value
		}
	}
	return ""
}

type Image struct {
	awsec2.Image
	err error
}
