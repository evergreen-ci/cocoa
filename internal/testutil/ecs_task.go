package testutil

import (
	"os"
	"strings"

	"github.com/evergreen-ci/utility"
)

// NewTaskDefinitionFamily makes a new test family for a task definition with a
// common prefix, the given name, and a random string.
func NewTaskDefinitionFamily(name string) string {
	return strings.Join([]string{os.Getenv("AWS_ECS_TASK_DEFINITION_PREFIX"), "cocoa", strings.ReplaceAll(name, "/", "-"), utility.RandomString()}, "-")
}
