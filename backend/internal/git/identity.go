package git

import (
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetDefaultSignature returns the default author/committer signature for operations.
// In a real application, this should retrieve user configuration.
func GetDefaultSignature() *object.Signature {
	return &object.Signature{
		Name:  "User",
		Email: "user@example.com",
		When:  time.Now(),
	}
}
