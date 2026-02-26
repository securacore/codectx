package cmdx

// TagType identifies the kind of a CMDX AST node.
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
	TagKV
	TagParams
	TagEndpoint
	TagReturns
	TagDef
	TagNote
	TagWarn
	TagTip
	TagImportant
	TagMeta
	TagDict
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
	TagKV:            "KV",
	TagParams:        "PARAMS",
	TagEndpoint:      "ENDPOINT",
	TagReturns:       "RETURNS",
	TagDef:           "DEF",
	TagNote:          "NOTE",
	TagWarn:          "WARN",
	TagTip:           "TIP",
	TagImportant:     "IMPORTANT",
	TagMeta:          "META",
	TagDict:          "DICT",
	TagRaw:           "RAW",
}

// String returns the CMDX tag name for this type.
func (t TagType) String() string {
	if name, ok := tagNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

// Document represents a parsed CMDX document.
type Document struct {
	Version string
	Dict    *Dictionary
	Meta    map[string]string
	Body    []Node
}

// Dictionary maps $N reference tokens to expanded string values.
type Dictionary struct {
	Entries []DictEntry
	index   map[string]string // reverse lookup: value -> "$N" (encoder use)
}

// Lookup returns the value for a dictionary index, or empty string if not found.
func (d *Dictionary) Lookup(idx int) (string, bool) {
	if d == nil || idx < 0 || idx >= len(d.Entries) {
		return "", false
	}
	return d.Entries[idx].Value, true
}

// DictEntry is a single dictionary mapping.
type DictEntry struct {
	Index int
	Value string
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
	Language string       // CODE blocks
	Level    int          // Headings (1-6)
	Items    []KVItem     // KV blocks
	Params   []ParamItem  // PARAMS blocks
	Cells    [][]string   // TABLE body rows
	Headers  []string     // TABLE header row
	URL      string       // LINK, IMG
	Display  string       // LINK, IMG
	Callout  string       // NOTE, WARN, TIP, IMPORTANT
	Endpoint *EndpointDef // ENDPOINT blocks
	Returns  []ReturnDef  // RETURNS blocks
}

// KVItem represents a key-value documentation entry.
type KVItem struct {
	Key         string
	Type        string
	Description string
}

// ParamItem represents a parameter documentation entry.
type ParamItem struct {
	Name        string
	Type        string
	Required    bool // R=true, O=false
	Description string
}

// EndpointDef represents a REST API endpoint.
type EndpointDef struct {
	Method string // GET, POST, PUT, DELETE, PATCH
	Path   string
}

// ReturnDef represents a return status description.
type ReturnDef struct {
	Status      string
	Description string
}

// EncoderOptions configures the encoding process.
type EncoderOptions struct {
	MaxDictEntries     int  // Default: 50
	MinStringLength    int  // Default: 10
	MinFrequency       int  // Default: 2
	EnableDomainBlocks bool // Default: true
	PreserveMeta       bool // Default: true
}

// DefaultEncoderOptions returns sensible defaults.
func DefaultEncoderOptions() EncoderOptions {
	return EncoderOptions{
		MaxDictEntries:     50,
		MinStringLength:    10,
		MinFrequency:       2,
		EnableDomainBlocks: true,
		PreserveMeta:       true,
	}
}

// Stats holds compression analysis results.
type Stats struct {
	OriginalBytes   int
	CompressedBytes int
	ByteSavings     float64 // percentage
	DictEntries     int
	DictSavings     int // bytes saved by dictionary alone
	DomainSavings   int // bytes saved by domain blocks alone
	EstTokensBefore int // rough estimate: bytes / 4
	EstTokensAfter  int
	TokenSavings    float64 // percentage
}
