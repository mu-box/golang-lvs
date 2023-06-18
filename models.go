package lvs

import (
// "encoding/json"
// "fmt"
// "strings"
)

type (
	ToJson interface {
		ToJson() ([]byte, error)
	}

	FromJson interface {
		FromJson([]byte) error
	}
)
