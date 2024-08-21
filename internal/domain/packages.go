package domain

type PackageInfo struct {
	Name    string
	Version string
	Repo    string
}

type InstalledPackagesResult struct {
	Packages []PackageInfo `json:"packages"`
}

type CheckUpdateResult struct {
	UpdatesAvailable  bool
	UpdatablePackages []PackageInfo
}
