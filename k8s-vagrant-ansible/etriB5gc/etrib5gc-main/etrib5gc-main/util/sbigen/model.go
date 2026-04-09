package main

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"strings"
)

type DataModel struct {
	id         string
	goType     string
	isArray    bool
	isMap      bool
	isExternal bool
	properties map[string]Property
	schema     *base.Schema
	enum       *Enum
}

func (m *DataModel) TypeName() string {
	goType := strings.Title(m.goType)
	if m.isArray {
		return "ArrayOf" + goType
	}
	if m.isMap {
		return "MapOf" + goType
	}
	return goType
}

func (m *DataModel) writeGoType(prefix string) string {

	/*
		if m.enum != nil {
			return prefix + m.id
		}
	*/
	prePrefix := ""
	if m.isArray {
		prePrefix = "[]"
	} else if m.isMap {
		prePrefix = "map[string]"
	}

	if m.isExternal || isPrimitive(m.goType) {
		return prePrefix + m.goType
	} else {
		return prePrefix + prefix + m.goType
	}
}

type Enum struct {
	enumType string
	values   []string
}

type Property struct {
	id       string
	required bool
	m        *DataModel
}

func (p *Property) hasEmptyValue() bool {
	if p.m.isArray {
		return true
	}

	if p.m.isMap {
		return true
	}

	if p.m.goType == "string" {
		return true
	}

	return false
}

func (p *Property) writeTag() string {
	if p.required {
		return fmt.Sprintf("`json:\"%s\"`", p.id)
	} else {
		return fmt.Sprintf("`json:\"%s,omitempty\"`", p.id)
	}
}
