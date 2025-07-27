package assets

import (
	"embed"
	"io/fs"
)

// WorkspaceTemplate contains all workspace template files
//
//go:embed all:workspace-template
var WorkspaceTemplate embed.FS

// GetWorkspaceTemplateFS returns the embedded filesystem for workspace templates
func GetWorkspaceTemplateFS() fs.FS {
	subFS, err := fs.Sub(WorkspaceTemplate, "workspace-template")
	if err != nil {
		// This should never happen with a valid embed
		panic("failed to create sub-filesystem for workspace-template: " + err.Error())
	}
	return subFS
}
