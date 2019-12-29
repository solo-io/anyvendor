package protodep

// need to reenable this once the functionality to vendor the protos is enabled
//-go:generate bash generate.sh

const (
	// default directory into which proto, and other files will be vendored.
	// Originally this was meant to be the vendor directory, but clashes with the go vendor directory
	// meant it would be easier for this to inhabit it's own folder
	// See this section for more info: https://tip.golang.org/doc/go1.14#go-command
	// This tool should not force to projects to build using vendor.
	DefaultDepDir = ".proto_vendor"
)
