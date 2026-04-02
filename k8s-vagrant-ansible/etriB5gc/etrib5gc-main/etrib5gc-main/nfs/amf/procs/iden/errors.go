package iden

import (
	"fmt"
)

var (
	ErrMismatchedIdType error = fmt.Errorf("UE send an identity with  mismatched id type")
	ErrNoResponse       error = fmt.Errorf("UE does not send IdentityResponse")
)
