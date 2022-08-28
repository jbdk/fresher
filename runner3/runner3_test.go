package runner3

import (
	"testing"

	"github.com/c9845/fresher/config"
)

func TestShouldRebuild(t *testing.T) {
	//Configure, so we know what extensions to rebuild on.
	config.UseDefaults()

	//The event name we use in Start() to build and run the binary for the first time
	//when fresher starts.
	eventName := "/"
	if !shouldRebuild(eventName) {
		t.Fatal("shouldRebuild should be return 'true' on '/' event to start building on fresher starting")
	}

	//Handle extension to rebuild on.
	eventName = "path/to/hello.go"
	if !shouldRebuild(eventName) {
		t.Fatal("shouldRebuild should be return 'true' on files ending in .go event to start building on fresher starting")
	}

	//Handle extension to skip rebuilding.
	eventName = "path/to/hello.html"
	if shouldRebuild(eventName) {
		t.Fatal("shouldRebuild should be return 'false' on files ending in .html event to start building on fresher starting")
	}
}
