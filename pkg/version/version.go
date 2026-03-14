/*
Package version holds the build version for the standalone commodore binary.
The Version variable is overridden at link time with:

	go build -ldflags="-X github.com/OctalMesh/Commodore/pkg/version.Version=<v>"
*/
package version

var Version = "dev"
