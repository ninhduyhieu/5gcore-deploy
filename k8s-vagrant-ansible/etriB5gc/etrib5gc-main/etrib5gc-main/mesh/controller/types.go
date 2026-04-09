package controller

import (
	"fmt"
	"strings"
)

const (
	LB_TYPE_RANDOM uint8 = iota
	LB_TYPE_ROUND_ROBIN
)

type Selectors map[string]string

func (s Selectors) String() string {
	items := []string{}
	for k, v := range s {
		items = append(items, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(items, ", ")
}
