package pubsub

import (
	"net/url"

	"robinplatform.dev/internal/project"
)

// ID for an App running in the project. The following
func AppProcessLogs(category string, key string) TopicId {
	name, err := project.GetProjectName()
	if err != nil {
		panic(err)
	}

	name = url.PathEscape(name)

	if category != "" {
		category = "/" + url.PathEscape(category)
	}

	return TopicId{
		Category: "@robin/logs/@robin/app/" + name + category,
		Name:     key,
	}
}
