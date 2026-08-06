package pragma

import "fmt"

type Property string

const (
	PageSizeProperty = "page_size"
	TokenProperty    = "token"
	ReadMetadata     = "read_metadata"
)

func Parse(s string) (Property, error) {
	k := Property(s)

	if s == "" {
		return k, fmt.Errorf("PRAGMA parse: empty string")
	}

	switch k {
	case PageSizeProperty, TokenProperty, ReadMetadata:
		return k, nil
	}

	return k, fmt.Errorf("PRAGMA parse: could not parse %q", s)
}
