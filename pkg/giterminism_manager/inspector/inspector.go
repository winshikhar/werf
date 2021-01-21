package inspector

type Inspector struct {
	giterminismConfig giterminismConfig
	sharedContext     sharedContext
}

func NewInspector(giterminismConfig giterminismConfig, sharedContext sharedContext) Inspector {
	return Inspector{giterminismConfig: giterminismConfig, sharedContext: sharedContext}
}

type giterminismConfig interface {
	IsConfigGoTemplateRenderingEnvNameAccepted(envName string) (bool, error)
	IsConfigStapelFromLatestAccepted() bool
	IsConfigStapelGitBranchAccepted() bool
	IsConfigStapelMountBuildDirAccepted() bool
	IsConfigStapelMountFromPathAccepted(fromPath string) (bool, error)
	IsConfigDockerfileContextAddFileAccepted(relPath string) (bool, error)
}

type sharedContext interface {
	LooseGiterminism() bool
}
