package preorders

import (
	"fmt"
	"strconv"
)

type optionalBool struct {
	set   bool
	value bool
}

func (b *optionalBool) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("must be true or false")
	}
	b.value = parsed
	b.set = true
	return nil
}

func (b *optionalBool) String() string {
	if !b.set {
		return ""
	}
	return strconv.FormatBool(b.value)
}

func (b *optionalBool) IsBoolFlag() bool {
	return true
}
