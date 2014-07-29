package imgstore

import (
	"time"
)

// implementation detail of the image storage, but not signable data
type ImageOnDisk struct {
	Image

	// for sorting the pulled/loaded image
	Installed time.Time
	PreviousHashes []string
}

// minimal signed struct
type Image struct {
	// Docker API version? is this a net-new versioning?
	Schema SchemaVersion

	// Each image has exactly one name.
	// Images are self-describing: their manifest includes their name.
	//
	// An image name is made of 2 parts:
	// 	- A repository, for example 'shykes/myapp'.
	//	- A tag, for example 'v2'.
	// All image names are part of a globally unique namespace, which is
	// enforced by a common federated naming system. This allows using
	// repositories as a trust unit: if you know how much you trust the
	// entity behind a certain repository name, you can reliably enforce
	// that level of trust for all images under that name, because Docker
	// guarantees that they were published by the same entity.
	Name struct {
		Repository	string
		Tag		string

		// shykes/myapp:v2	[hash 4242424242]
		// previousTag: v1
		//
		// shykes/myapp:v1	[hash 4343434343]
		// shykes/myapp:v1	[hash 4141414141]
		PreviousTag	string
		// have a Comparable? :-)
		Version Version
	}

	Hash string

	Author      string
	Created     time.Time
	Description string

	// more of a step towards #6805
	//Signature() Signature

	// A map of commands exposed by the container by default.
	// By convention the default command should be called 'main'.
	// 'docker start' or 'docker run', when not given any arguments, will start that command.
	Commands map[string]struct {
		// The command to execute
		Path string
		Args string

		// Which hardware architectures and OS can this command run on?
		Arch []Arch
		OS   []OS

		// The user id under which to run this command
		User      string
		Env       []string
		Tty       bool
		OpenStdin bool
	}

	// The filesystem layer to mount at /
	FSLayer LayerHash

	/*
	FS []struct {
		Layer LayerHash
		Op    LayerOp
		Dst   string
	}
	*/


	OnBuild []string

	/*
	// ONBUILD?
	Triggers map[TriggerType][]string
	*/
}

func GetLayer(lh LayerHash) (Layer, error) {
	// ...
}

type Layer interface {
	RootFs() string
	Hash() string

	// more of a step towards #6805
	//Signature() Signature
}

type Signature interface {
}

type SchemaVersion uint32

const (
	Schema_1_1 SchemaVersion = iota + 1
)

type Arch string

const (
	X86    Arch = "x86"
	X86_64 Arch = "x86_64"
	Arm    Arch = "arm"
)

type OS string

const (
	Linux   OS = "linux"
	Darwin  OS = "darwin"
	Windows OS = "windows"
)

type TriggerType string

const (
	OnBuild TriggerType = "onbuild"
)

type LayerHash struct {
	Type  HashType
	Value string
}

type HashType string

// "tarsum+sha256"
const (
	Tarsum1 HashType = "tarsum"
	Tarsum2 HashType = "tarsum2"
)

type LayerOp string

const (
	OpUnpack  LayerOp = "unpack"
	OpCopy    LayerOp = "copy"
	OpMount   LayerOp = "mount"
	OpMountRO LayerOp = "mountro"
)
