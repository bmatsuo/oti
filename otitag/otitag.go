/*
tags used by oti.
*/
package otitag

type OTITag string

var Tags = []OTITag{
	ResourceId,
	Target,
	Created,
}

const (
	ResourceId OTITag = "ResourceId" // a unique identifier for the resource.
	Target     OTITag = "Target"     // an identifier that groups resources.
	Created    OTITag = "Created"    // an timestamp in RFC3339 format.
)

// tags present only on instances
var InstanceTags = []OTITag{
	IImageId,
}

const (
	IImageId OTITag = "ImageId"
)

// returns all tags; Tags, InstanceTags, etc.
func AllTags() []OTITag {
	return append(append([]OTITag{},
		Tags...),
		InstanceTags...)
}
