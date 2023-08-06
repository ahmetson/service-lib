package service

const (
	// SourceName of this type should be listed within the controllers in the config
	SourceName = "source"
	// DestinationName of this type should be listed within the controllers in the config
	DestinationName = "destination"
)

type Proxy struct {
	Url       string
	Instances []Instance
}
