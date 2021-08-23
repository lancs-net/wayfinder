module github.com/lancs-net/wayfinder

go 1.14

require (
	github.com/containerd/containerd v1.4.3
	github.com/containerd/continuity v0.0.0-20201208142359-180525291bb7 // indirect
	github.com/lancs-net/netns v0.5.4
	github.com/google/go-containerregistry v0.3.0
	github.com/moby/moby v20.10.1+incompatible
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/muesli/termenv v0.7.4
	github.com/novln/docker-parser v1.0.0
	github.com/opencontainers/runc v1.0.0-rc92
	github.com/opencontainers/runtime-spec v1.0.3-0.20200728170252-4d89ac9fbff6
	github.com/otiai10/copy v1.4.2
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.1
	github.com/tidwall/gjson v1.6.7
	golang.org/x/sys v0.0.0-20201201145000-ef89a241ccb3
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/moby/sys/mountinfo => github.com/moby/sys/mountinfo v0.2.0
