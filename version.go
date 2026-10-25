package quickiedata

import "runtime/debug"

// versionUnknown is the version reported for builds that carry none.
const versionUnknown = "unknown"

// libraryModulePath is this module, as it appears in build info.
const libraryModulePath = "github.com/rohfle/quickiedata"

// Version returns the release version of this library, for example "v0.2.0",
// or "unknown" for a build that carries no version, such as a local one.
func Version() string {
	return getVersion()
}

// getVersion returns the quickiedata release version, or versionUnknown for
// local builds, which report no version or "(devel)".
func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return versionUnknown
	}
	return versionFrom(info)
}

// versionFrom reads the quickiedata version out of build info: the main module
// version when quickiedata was built as the main module, otherwise the version
// of the dependency, following a replacement.
func versionFrom(info *debug.BuildInfo) string {
	if info.Main.Path == libraryModulePath && isReleaseVersion(info.Main.Version) {
		return info.Main.Version
	}
	for _, module := range info.Deps {
		if module.Path != libraryModulePath {
			continue
		}
		if module.Replace != nil {
			module = module.Replace
		}
		if isReleaseVersion(module.Version) {
			return module.Version
		}
	}
	return versionUnknown
}

// isReleaseVersion reports whether a module version identifies a release,
// rather than the empty version or the "(devel)" of a local build.
func isReleaseVersion(version string) bool {
	return version != "" && version != "(devel)"
}
