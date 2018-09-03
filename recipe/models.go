package recipe

type Executor interface {
	ExecuteRecipe(appID, stagingGUID, completionCallback, eiriniAddr, providedBuildpacksJSON string) error
}

//go:generate counterfeiter . Uploader
type Uploader interface {
	Upload(guid, path string) error
	UploadCache(appGUID, stagingGUID, path string) error
}

//go:generate counterfeiter . Installer
type Installer interface {
	Install(appID, targetDir string) error
}

//go:generate counterfeiter . Commander
type Commander interface { //todo
	Exec(cmd string, args ...string) error
}
