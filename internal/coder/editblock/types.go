package editblock

import "regexp"

type EditBlockCoder struct {
	Fence          [2]string
	RootPath       string
	ValidFilenames []string
}

type EditBlock struct {
	Filename string
	Original string
	Updated  string
}

type ShellCommand struct {
	Command string
}

type EditResult struct {
	Edit  *EditBlock
	Shell *ShellCommand
	Err   error
}

const (
	searchMarker  = "<<<<<<< SEARCH"
	divider       = "======="
	replaceMarker = ">>>>>>> REPLACE"
)

var (
	headRe    = regexp.MustCompile(`^<{5,9} SEARCH\s*$`)
	dividerRe = regexp.MustCompile(`^={5,9}\s*$`)
	replaceRe = regexp.MustCompile(`^>{5,9} REPLACE\s*$`)
)
