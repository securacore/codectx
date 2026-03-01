package md

// TagType identifies the kind of an AST node.
type TagType int

const (
	TagH1 TagType = iota
	TagH2
	TagH3
	TagH4
	TagH5
	TagH6
	TagHR
	TagBR
	TagP
	TagBQ
	TagBold
	TagItalic
	TagBoldItalic
	TagCode
	TagCodeBlock
	TagStrikethrough
	TagLink
	TagImage
	TagUL
	TagOL
	TagTable
	TagRaw
)

var tagNames = map[TagType]string{
	TagH1:            "H1",
	TagH2:            "H2",
	TagH3:            "H3",
	TagH4:            "H4",
	TagH5:            "H5",
	TagH6:            "H6",
	TagHR:            "HR",
	TagBR:            "BR",
	TagP:             "P",
	TagBQ:            "BQ",
	TagBold:          "B",
	TagItalic:        "I",
	TagBoldItalic:    "BI",
	TagCode:          "C",
	TagCodeBlock:     "CODE",
	TagStrikethrough: "S",
	TagLink:          "LINK",
	TagImage:         "IMG",
	TagUL:            "UL",
	TagOL:            "OL",
	TagTable:         "TABLE",
	TagRaw:           "RAW",
}

// String returns the tag name for this type.
func (t TagType) String() string {
	if name, ok := tagNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

// Node represents a single element in the document AST.
type Node struct {
	Tag      TagType
	Content  string
	Attrs    NodeAttrs
	Children []Node
}

// NodeAttrs holds tag-specific attributes.
type NodeAttrs struct {
	Language string     // CODE blocks
	Cells    [][]string // TABLE body rows
	Headers  []string   // TABLE header row
	URL      string     // LINK, IMG
	Display  string     // LINK, IMG
}

// Stats holds compression analysis results.
type Stats struct {
	OriginalBytes   int
	CompressedBytes int
	ByteSavings     float64 // percentage
	EstTokensBefore int     // token count via o200k_base BPE tokenizer
	EstTokensAfter  int     // token count via o200k_base BPE tokenizer
	TokenSavings    float64 // percentage
}
