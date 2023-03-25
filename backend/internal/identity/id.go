package identity

import (
	"fmt"
	"net/url"
	"path"
)

type Id struct {
	// The category of the ID. This is used by Robin internally to namespace
	// IDs, and should always be a valid URL path beginning with `/`. All
	// user generated input should be escaped first.
	//
	// The following path formats are currently used:
	// - /app/{project}/{app-id} - the category for an app's spawned processes
	// - /app/{project} - the category for a project's spawned apps
	// - /logs/{app-category} - logs for an app with a certain category
	// - /topics - meta category for information about topics
	Category string `json:"category"`
	// The identifier used to refer to an object. This is not cleaned, and has no
	// guarantees about formatting.
	Key string `json:"key"`
}

func (id Id) String() string {
	// The '#' character gets path escaped to '%23', so there's no way it can be in the category,
	// making it possible to go back and forth between this format and the struct.
	return fmt.Sprintf(
		"%s#%s",
		id.Category,
		id.Key,
	)
}

// Cleans inputs and then creates a category from them. If you have a valid category already,
// ust path.Join to combine it with another category.
func Category(ids ...string) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, url.PathEscape(id))
	}

	return "/" + path.Join(parts...)
}
