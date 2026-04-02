package main

import (
	"fmt"
	"strings"
)

type RequestBody struct {
	required bool
	content  DataModel
	desc     string
}

type Parameter struct {
	id       string
	in       string
	desc     string
	required bool
	m        DataModel
}

func (p *Parameter) hasEmptyValue() bool {
	return p.m.isArray || p.m.isMap || p.m.goType == "string"
}

func (p *Parameter) isPrimitive() bool {
	if p.m.isArray || p.m.isMap {
		return false
	}
	return isPrimitive(p.m.goType)
}

func (p *Parameter) getName(capitalize bool) string {
	if len(p.id) == 0 {
		return ""
	}

	parts := strings.FieldsFunc(p.id, func(c rune) bool {
		return c == ' ' || c == '-' || c == '_'
	})
	out := ""
	for i, part := range parts {
		if i == 0 {
			if part[0] == '5' {
				out = "five" + part[1:]
			} else if part[0] == '3' {
				out = "three" + part[1:]
			} else {
				out = strings.ToLower(string(part[0])) + part[1:]
			}
			if capitalize {
				out = strings.Title(out)
			}
		} else {
			out = out + strings.Title(part)
		}
	}
	return out
}

func (p *Parameter) getTypeDefinition() (def string) {
	def = p.m.writeGoType("models.")
	if p.m.isArray || p.m.isMap || p.m.goType == "string" {
		return
	}
	if p.required {
		if isPrimitive(p.m.goType) {
			return
		}
	}
	def = "*" + def
	return
}
func (p *Parameter) isStringType() bool {
	if p.m.isArray || p.m.isMap {
		return false
	}
	return p.m.goType == "string"
}

func (p *Parameter) TypeName() string {
	return p.m.TypeName()
}

func (p *Parameter) getDefinition(capitalize bool) string {
	return fmt.Sprintf("%s %s", p.getName(capitalize), p.getTypeDefinition())
}

func (p *Parameter) getDefinedName(prefix string) string {
	if len(prefix) > 0 {
		return prefix + p.getName(true)
	}
	return p.getName(false)

}
func (p *Parameter) writeParamAdd(prefix string) string {
	lines := []string{}

	varStr := p.getDefinedName(prefix)

	pointer := ""
	if def := p.getTypeDefinition(); def[0] == '*' { //is var definition has a pointer?
		pointer = "*"
	}
	convertStr := varStr

	if !p.isStringType() {
		convertStr = fmt.Sprintf("models.%sToString(%s%s)", p.TypeName(), pointer, varStr)
	}
	var paramAdder string

	if p.in != "path" {
		if p.in == "header" {
			paramAdder = fmt.Sprintf("request.AddHeader(\"%s\", %s)", p.id, convertStr)
		} else {
			paramAdder = fmt.Sprintf("request.AddParam(\"%s\", %s)", p.id, convertStr)
		}
	}
	if p.required { //required param
		lines = append(lines, paramAdder)
	} else { //optional param
		if p.hasEmptyValue() {
			lines = append(lines, fmt.Sprintf("if len(%s) > 0 {", varStr))
			lines = append(lines, paramAdder)
			lines = append(lines, fmt.Sprintf("}"))
		} else {
			lines = append(lines, fmt.Sprintf("if %s != nil {", varStr))
			lines = append(lines, paramAdder)
			lines = append(lines, fmt.Sprintf("}"))
		}
	}
	return strings.Join(lines, "\n")
}

// write code for checking nil value of a parameter and add to the request
func (p *Parameter) writeParamCheck(prefix string) string {
	varStr := p.getDefinedName(prefix)

	if p.required { //required param
		if p.hasEmptyValue() {
			return fmt.Sprintf("if len(%s) == 0 {\nerr = fmt.Errorf(\"%s is required\")\nreturn\n}", varStr, p.id)
		} else if !p.isPrimitive() { //struct type
			return fmt.Sprintf("if %s == nil {\nerr = fmt.Errorf(\"%s is required\")\nreturn\n}", varStr, p.id)
		}
	}
	return ""
}

func (p *Parameter) writeExtractParam(prefix string) string {
	out := []string{}

	var varStr string

	paramName := p.getName(false)

	out = append(out, fmt.Sprintf("\n// read '%s'", p.id))

	inStruct := len(prefix) > 0
	if inStruct {
		varStr = prefix + p.getName(true)
	} else {
		//write parameter definition
		out = append(out, fmt.Sprintf("var %s", p.getDefinition(false)))
		varStr = paramName
	}
	//get parameter string from request
	var convertFn, tempStr string
	getterFn := "ctx.Param"
	if p.in == "header" {
		getterFn = "ctx.Header"
	}
	if p.isStringType() {
		out = append(out, fmt.Sprintf("%s = %s(\"%s\")", varStr, getterFn, p.id))
		tempStr = varStr
	} else {
		tempStr = paramName + "Str"
		convertFn = fmt.Sprintf("models.%sFromString(%s)", p.TypeName(), tempStr)
		out = append(out, fmt.Sprintf("%s := %s(\"%s\")", tempStr, getterFn, p.id))
	}

	//check for non empty string and convert if neccessary
	if p.required {
		errBlock := fmt.Sprintf("\nctx.WriteResponse(400, models.CreateProblemDetails(400, \"%s is required\"), nil)\nreturn\n", p.id)
		out = append(out, fmt.Sprintf("if len(%s) == 0 {%s}\n", tempStr, errBlock))
		//convert from string
		if !p.isStringType() {
			errBlock = fmt.Sprintf("\nctx.WriteResponse(400, models.CreateProblemDetails(400, fmt.Sprintf(\"parse %s failed: %%+v\", err)), nil)\nreturn\n", p.id)
			out = append(out, fmt.Sprintf("if %s, err = %s; err != nil {%s}\n", varStr, convertFn, errBlock))
		}
	} else {
		//convert from string
		if !p.isStringType() {
			errBlock := fmt.Sprintf("\nctx.WriteResponse(400, models.CreateProblemDetails(400, fmt.Sprintf(\"parse %s failed: %%+v\", err)), nil)\nreturn\n", p.id)
			// optional primitive param needs a tmp var to keep converted value
			parsingBlock := ""
			convertVar := varStr
			if p.isPrimitive() {
				convertVar = paramName + "Tmp"
				parsingBlock = fmt.Sprintf("var %s %s\n", convertVar, p.m.goType)
			}

			parsingBlock = fmt.Sprintf("%sif %s, err = %s; err != nil {%s}\n", parsingBlock, convertVar, convertFn, errBlock)

			if p.isPrimitive() {
				parsingBlock = fmt.Sprintf("%s%s=&%s\n", parsingBlock, varStr, convertVar)
			}

			out = append(out, fmt.Sprintf("if len(%s) > 0 {\n%s}\n", tempStr, parsingBlock))
		}
	}

	return strings.Join(out, "\n")
}

func (p *Parameter) stringConvertFn(prefix string) string {

	var pStr string
	if len(prefix) > 0 {
		pStr = prefix + p.getName(true)
	} else {
		pStr = p.getName(false)
	}
	if !p.isStringType() {

		if def := p.getTypeDefinition(); def[0] == '*' { //is var definition has a pointer?
			return fmt.Sprintf("models.%sToString(*%s)", p.TypeName(), pStr)
		} else {
			return fmt.Sprintf("models.%sToString(%s)", p.TypeName(), pStr)
		}
	}

	return pStr
}

type Operation struct {
	path        string
	pathTmpl    string
	pathParams  []string
	method      string
	id          string
	summary     string
	desc        string
	parameters  map[string]Parameter
	requestBody *RequestBody

	//for responses
	sHeaders          []string //headers for success response
	eHeaders          []string //headers for error response
	errorModel        *DataModel
	successModel      *DataModel
	errorCodes        []string //with Defined Error
	problemCodes      []string //with ProblemDetails
	successCode       string   //success code with a response (only one)
	emptySuccessCodes []string //success code with empty response
}

func (o *Operation) writeMethod() string {
	switch strings.ToLower(o.method) {
	case "get":
		return "http.MethodGet"
	case "post":
		return "http.MethodPost"
	case "put":
		return "http.MethodPut"
	case "delete":
		return "http.MethodDelete"
	case "patch":
		return "http.MethodPatch"
	}
	return "http.MethodGet"
}
