package secmode

import (
	"fmt"
)

var (
	ErrNoResponse error = fmt.Errorf("UE did not send a SecurityModeComplete")
)
