package testutil

import (
	"fmt"
	"os"
	"path"

	"github.com/evergreen-ci/utility"
)

const projectName = "cocoa"

// NewSecretName creates a new test secret name with a common prefix, the given
// name, and a random string.
func NewSecretName(name string) string {
	return fmt.Sprint(path.Join(os.Getenv("AWS_SECRET_PREFIX"), projectName, name, utility.RandomString()))
}
