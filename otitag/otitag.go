/*
tags used by oti.
*/
package otitag

type OTITag string

var Tags = []OTITag{
	ResourceId,
	SessionId,
	Created,
}

const (
	ResourceId OTITag = "ResourceId" // a unique identifier for the resource.
	SessionId  OTITag = "SessionId"  // an identifier that groups oti resources.
	Created    OTITag = "Created"    // an timestamp in RFC3339 format.
)

// tags present only on instances
var InstanceTags = []OTITag{
	IImageId,
}

const (
	IImageId OTITag = "Instance.ImageId" // ResourceId of a machine image
)

// tags present only on instances
var ImageTags = []OTITag{
	ImType,
}

const (
	ImType OTITag = "Image.Type" // ResourceId of a machine image
)

// returns all tags; Tags, InstanceTags, etc.
func AllTags() []OTITag {
	return concat(
		Tags,
		InstanceTags,
		ImageTags,
	)
}

func concat(ts ...[]OTITag) []OTITag {
	if len(ts) == 0 {
		return nil
	}

	var _ts []OTITag

	for i := range ts {
		_ts = append(_ts, ts[i]...)
	}

	return _ts
}
