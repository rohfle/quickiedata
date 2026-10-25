package quickiedata

import (
	"runtime/debug"
	"testing"
)

// TestVersionFrom checks the version reported for the builds quickiedata runs
// in: as the main module, as a dependency, and as a dependency with a
// replacement.
func TestVersionFrom(t *testing.T) {
	module := func(path, version string) *debug.Module {
		return &debug.Module{Path: path, Version: version}
	}

	tests := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{
			name: "main module with a release version",
			info: &debug.BuildInfo{Main: debug.Module{Path: libraryModulePath, Version: "v1.2.3"}},
			want: "v1.2.3",
		},
		{
			name: "main module during go test",
			info: &debug.BuildInfo{Main: debug.Module{Path: libraryModulePath, Version: "(devel)"}},
			want: versionUnknown,
		},
		{
			name: "dependency with a release version",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/app", Version: "(devel)"},
				Deps: []*debug.Module{module("github.com/spf13/cobra", "v1.10.2"), module(libraryModulePath, "v0.4.1")},
			},
			want: "v0.4.1",
		},
		{
			name: "dependency with a replacement",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/app"},
				Deps: []*debug.Module{{
					Path:    libraryModulePath,
					Version: "v0.4.1",
					Replace: module("example.com/fork", "v0.9.0"),
				}},
			},
			want: "v0.9.0",
		},
		{
			name: "dependency replaced by a local directory",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/app"},
				Deps: []*debug.Module{{
					Path:    libraryModulePath,
					Version: "v0.4.1",
					Replace: module("/src/quickiedata", ""),
				}},
			},
			want: versionUnknown,
		},
		{
			name: "quickiedata not in the build",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/app", Version: "v3.0.0"},
				Deps: []*debug.Module{module("github.com/spf13/cobra", "v1.10.2")},
			},
			want: versionUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := versionFrom(test.info); got != test.want {
				t.Errorf("expected %q, got %q", test.want, got)
			}
		})
	}
}
