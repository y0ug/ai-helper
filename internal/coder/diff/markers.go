package diff

const (
	SearchMarker    = "<<<<<<< SEARCH"
	ReplaceMarker   = ">>>>>>> REPLACE"
	SeparatorMarker   = "======="
)

type Section struct {
	Filename     string
	SearchBlock  string
	ReplaceBlock string
	Language     string
}
