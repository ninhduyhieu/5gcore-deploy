package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var log *logrus.Entry

func init() {
	log = logrus.WithFields(logrus.Fields{})
}

var _flags = []cli.Flag{
	cli.StringFlag{
		Name:  "spec, f",
		Usage: "Apis specification `FILE`",
	},
	cli.StringFlag{
		Name:        "sbi-pkg, s",
		Usage:       "Prefix of existing Golang SBI root package",
		Value:       "github.com/reogac/sbi",
		Destination: &_sbiPackage,
	},
	cli.StringFlag{
		Name:        "sbi-dir, d",
		Usage:       "Sbi root directory",
		Value:       "sbi",
		Destination: &_sbiRootDir,
	},
	cli.StringFlag{
		Name:        "app, a",
		Usage:       "Network function (aka. application) name",
		Destination: &_appName,
	},
	cli.StringFlag{
		Name:        "pkg, p",
		Usage:       "Golang service package name",
		Destination: &_pkgName,
	},
	cli.BoolFlag{
		Name:        "model-ow, w",
		Usage:       "Model overwriting enable/disable",
		Destination: &_modelOverwrite,
	},
	cli.StringFlag{
		Name:  "ext-pkg, l",
		Usage: "list of external package to import; separate by commas",
	},
}

var models map[string]DataModel
var _sbiRootDir string
var _sbiPackage string
var _modelOverwrite bool
var _appName string
var _pkgName string
var _apiRootPath string
var _externalPackages []string

func main() {
	app := &cli.App{
		Name:   "sbigen",
		Usage:  "Sbi data models and serivce APIs generator",
		Action: action,
		Flags:  _flags,
	}

	if err := app.Run(os.Args); err != nil {
		log.Errorf(err.Error())
		return
	}
}

func action(c *cli.Context) error {
	var specDat []byte
	specFileName := c.String("spec")

	if len(specFileName) == 0 {
		return fmt.Errorf("Specification file is missing")
	}

	if len(_appName) == 0 {
		return fmt.Errorf("Application name is missing")
	}
	if len(_pkgName) == 0 {
		return fmt.Errorf("Package name is missing")
	}
	var tmpStr string
	var err error
	//check for existance of SBI root dir
	if _, err = os.Stat(_sbiRootDir); os.IsNotExist(err) {
		return fmt.Errorf("Root Sbi directory not exists: %s", _sbiRootDir)
	}

	//read spec file
	if _, err = os.Stat(specFileName); os.IsNotExist(err) { //spec file not in current dir
		specFileName = fmt.Sprintf("%s/specs/%s", _sbiRootDir, specFileName)
	}
	if specDat, err = os.ReadFile(specFileName); err != nil {
		return fmt.Errorf("Fail to read specification file: %+v", err)
	}

	//check/create app dir and pkg dir in the sbi root directory
	tmpStr = fmt.Sprintf("%s/apis/%s", _sbiRootDir, _appName)

	if _, err = os.Stat(tmpStr); os.IsNotExist(err) {
		if err = os.Mkdir(tmpStr, 0755); err != nil {
			return fmt.Errorf("Fail to create application directory %s: %+v", tmpStr, err)
		}
	}

	tmpStr = fmt.Sprintf("%s/apis/%s/%s", _sbiRootDir, _appName, _pkgName)
	if _, err = os.Stat(tmpStr); !os.IsNotExist(err) {
		if err = os.RemoveAll(tmpStr); err != nil {
			return fmt.Errorf("Fail to remove package directory %s: %+v", tmpStr, err)
		}
	}
	if err = os.Mkdir(tmpStr, 0755); err != nil {
		return fmt.Errorf("Fail to create package directory %s: %+v", tmpStr, err)
	}

	//check/create models directory in the sbi root director
	tmpStr = fmt.Sprintf("%s/models", _sbiRootDir)
	if _, err = os.Stat(tmpStr); os.IsNotExist(err) {
		if err = os.Mkdir(tmpStr, 0755); err != nil {
			return fmt.Errorf("Fail to create Sbi models directory %s: %+v", tmpStr, err)
		}
	}
	//read external packages
	_externalPackages = strings.Split(c.String("ext-pkg"), ",")
	//read specification
	return readSpec(specDat)
}

func readSpec(specDat []byte) error {
	config := &datamodel.DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: true,
		BasePath:              fmt.Sprintf("%s/specs", _sbiRootDir),
	}

	// create a new document from specification bytes
	document, err := libopenapi.NewDocumentWithConfiguration(specDat, config)

	// if anything went wrong, an error is thrown
	if err != nil {
		return fmt.Errorf("cannot parse OpenApi document: %+v", err)
	}

	// because we know this is a v3 spec, we can build a ready to go model from it.
	v3Model, errors := document.BuildV3Model()

	// if anything went wrong when building the v3 model, a slice of errors will be returned
	if len(errors) > 0 {
		for i := range errors {
			log.Errorf("error: %+v", errors[i])
		}
		return fmt.Errorf("cannot create v3 model from OpenApi document: %d errors reported",
			len(errors))
	}
	log.Infof("Title: %s, Desc: %s", v3Model.Model.Info.Title, v3Model.Model.Info.Description)
	// get a count of the number of paths and schemas.
	paths := v3Model.Model.Paths.PathItems
	schemas := v3Model.Model.Components.Schemas

	log.Infof("There are %d paths and %d schemas in the document", paths.Len(), schemas.Len())
	createModels(schemas)
	operations := readPaths(paths)
	for _, m := range models {
		writeModel(&m)
	}
	if servers := v3Model.Model.Servers; len(servers) > 0 {
		_apiRootPath = getRootPath(servers[0].URL)
	}
	writeApis(operations)
	return nil
}

func writeEnum(id string, enum *Enum) {
	prefix := _sbiRootDir + "/models/"

	filePath := prefix + makeModelName(id) + ".go"

	if !_modelOverwrite {
		if _, err := os.Stat(filePath); !os.IsNotExist(err) { //model file exists
			return
		}
	}

	f, _ := os.Create(filePath)
	defer f.Close()

	log.Infof("Define constant values for %s[%s]\n", id, enum.enumType)

	writeFileHeader(f)

	fmt.Fprintf(f, "package models\n")

	fmt.Fprintf(f, "type %s %s\n", id, enum.enumType)
	fmt.Fprintf(f, "// Define constant values for %s\n", id)
	fmt.Fprintf(f, "const (\n")
	for _, v := range enum.values {
		v = strings.Replace(v, " ", "", -1)
		def := fmt.Sprintf("%s_%s", id, v)
		def = strings.Replace(def, "-", "_", -1)
		def = strings.ToUpper(def)
		if enum.enumType == "string" {
			fmt.Fprintf(f, "\t %s %s = \"%s\"\n", def, id, v)
		} else {
			fmt.Fprintf(f, "\t %s %s = %s\n", def, id, v)
		}
	}

	fmt.Fprintf(f, ") \n")
}
func writeModel(m *DataModel) {

	if m.enum != nil {
		writeEnum(m.id, m.enum)
		return
	}

	if isPrimitive(m.goType) {
		return
	}

	if len(m.properties) == 0 {
		return
	}

	structName := m.goType

	prefix := _sbiRootDir + "/models/"
	filePath := prefix + structName + ".go"

	if !_modelOverwrite {
		if _, err := os.Stat(filePath); !os.IsNotExist(err) { //model file exists
			return
		}
	}

	log.Infof("Write model %s", structName)

	f, _ := os.Create(filePath)
	defer f.Close()
	writeFileHeader(f)
	fmt.Fprintf(f, "package models\n")

	fmt.Fprintf(f, "type %s struct {\n", structName)

	for _, p := range m.properties {
		if p.m == nil {
			log.Warnf("model %s has untype attribute %s", m.id, p.id)
			continue
		}

		goType := p.m.goType
		if p.m.enum != nil {
			goType = makeModelName(p.m.id)
		}
		/*
			if !isPrimitive(goType) {
				goType = makeModelName(goType)
			}
		*/
		if p.m.isArray {
			goType = "[]" + goType
		} else if p.m.isMap {
			goType = "map[string]" + goType
		}

		attr := makeModelName(p.id)

		if p.required || p.hasEmptyValue() {
			fmt.Fprintf(f, "\t %s\t%s\t%s\n", attr, goType, p.writeTag())
		} else {
			fmt.Fprintf(f, "\t %s\t*%s\t%s\n", attr, goType, p.writeTag())
		}
	}
	fmt.Fprintf(f, "}\n")
}

func getRootPath(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) > 1 {
		return strings.Join(parts[1:], "/")
	}
	return ""
}

func buildRoute(orgRoute string, params []string) (newRoute, routeTemplate string, foundParams []string, err error) {
	parts := strings.Split(orgRoute, "/")
	pathParams := []string{} //params in route
	newParts := []string{}   //for forming the route
	tmplParts := []string{}  //for forming route template

	for _, part := range parts {
		if param, ok := getPathParam(part); ok { //found a path param
			pathParams = append(pathParams, param)
			tmplParts = append(tmplParts, "%s")
			newParts = append(newParts, ":"+param)
		} else {
			newParts = append(newParts, part)
			tmplParts = append(tmplParts, part)
		}
	}
	if len(params) != len(pathParams) {
		err = fmt.Errorf("Num params not matched")
		return
	}

	for _, param := range pathParams {
		if inList(param, params) {
			foundParams = append(foundParams, param)
		} else {
			err = fmt.Errorf("path parameter %s not in operation parameter list", param)
			return
		}
	}

	newRoute = strings.Join(newParts, "/")
	routeTemplate = strings.Join(tmplParts, "/")
	return
}

func getPathParam(str string) (outParam string, ok bool) {
	if l := len(str); l >= 3 {
		if str[0] == '{' && str[l-1] == '}' {
			ok = true
			outParam = str[1 : l-1]
			return
		}
	}
	ok = false
	return
}
func writeImportPkgs(list []string) (str string) {
	newList := []string{}
	for _, item := range list {
		if len(item) > 0 {
			newList = append(newList, fmt.Sprintf("\"%s\"", item))
		}
	}
	return strings.Join(newList, "\n")
}
func writeApis(operations map[string]Operation) {
	pkgDir := fmt.Sprintf("%s/apis/%s/%s", _sbiRootDir, _appName, _pkgName)

	//write consumer.go
	fc, _ := os.Create(pkgDir + "/consumer.go")
	defer fc.Close()
	writeFileHeader(fc)
	fmt.Fprintf(fc, "package %s\n", _pkgName)
	importPkgs := []string{
		"fmt",
		"net/http",
		_sbiPackage,
		_sbiPackage + "/models",
	}
	importPkgs = append(importPkgs, _externalPackages...)

	fmt.Fprintf(fc, "import (\n%s\n)\n", writeImportPkgs(importPkgs))

	fmt.Fprintf(fc, "const (\n PATH_ROOT string = \"%s\"\n)\n", _apiRootPath)

	for _, op := range operations {
		writeConsumerApi(fc, &op)
	}

	//write producer.go
	fp, _ := os.Create(pkgDir + "/producer.go")
	defer fp.Close()
	writeFileHeader(fp)
	fmt.Fprintf(fp, "package %s\n", _pkgName)

	importPkgs = []string{
		_sbiPackage,
		_sbiPackage + "/models",
	}
	importPkgs = append(importPkgs, _externalPackages...)

	prodMethods := [][]byte{}
	prodMethodInfs := []string{}
	prodMethodImpls := []string{}
	useFmt := false
	useIo := false
	for _, op := range operations {
		buf := new(bytes.Buffer)
		inf, impl, f1, f2 := writeProducerApi(buf, &op)
		prodMethods = append(prodMethods, buf.Bytes())
		prodMethodInfs = append(prodMethodInfs, inf)
		prodMethodImpls = append(prodMethodImpls, impl)
		if f1 {
			useFmt = true
		}
		if f2 {
			useIo = true
		}
	}
	if useFmt {
		importPkgs = append(importPkgs, "fmt")
	}
	if useIo {
		importPkgs = append(importPkgs, "io")
	}

	fmt.Fprintf(fp, "import (\n%s\n)\n", writeImportPkgs(importPkgs))

	for _, method := range prodMethods {
		fp.Write(method)
		fmt.Fprintf(fp, "\n\n")
	}

	fmt.Fprintf(fp, "type Producer interface {\n%s\n}\n", strings.Join(prodMethodInfs, "\n"))

	//write producer.go
	fi, _ := os.Create(pkgDir + "/impl.go.bak")
	defer fi.Close()
	fmt.Fprintf(fi, "/*package your pkg\n\nimport (\n\t\"%s/models\"\n\t\"%s/apis/%s/%s\"\n)\n%s\n*/", _sbiPackage, _sbiPackage, _appName, _pkgName, strings.Join(prodMethodImpls, "\n\n"))

	//write routes.go
	fr, _ := os.Create(pkgDir + "/routes.go")
	defer fr.Close()
	writeFileHeader(fr)
	fmt.Fprintf(fr, "package %s\n", _pkgName)
	fmt.Fprintf(fr, "import (\n\"%s\"\n\"net/http\"\n)\n", _sbiPackage)

	fmt.Fprintf(fr, "var _routes = []sbi.Route[Producer]{\n")
	blocks := []string{}
	for _, op := range operations {
		blocks = append(blocks, fmt.Sprintf("{\nLabel:\"%s\",\nMethod:%s,\nPath:\"%s\",\nHandler:On%s,\n}", op.id, op.writeMethod(), op.path, op.id))
	}
	fmt.Fprint(fr, strings.Join(blocks, ",\n"))
	fmt.Fprintf(fr, ",\n}\n\n")
	fmt.Fprintf(fr, "func Routes() []sbi.Route[Producer] {\nreturn _routes\n}\n")
}

func writeFileHeader(f *os.File) {
	timeNow := time.Now().Format(time.UnixDate)
	fmt.Fprintf(f, "/*\nThis file is generated with a SBI APIs generator tool developed by ETRI\nGenerated at %v by TungTQ<tqtung@etri.re.kr>\nDo not modify\n*/\n\n", timeNow)
}

func writeProducerApi(f io.Writer, op *Operation) (methodInf, methodImpl string, useFmt bool, useIo bool) {

	//write function definition
	fmt.Fprintf(f, "func On%s(ctx sbi.RequestContext, prod Producer) {\n", op.id)

	var successModelName, errorModelName, bodyModelName string

	defineErr := false //should err be define?
	if op.requestBody != nil {
		defineErr = true
	} else {
		for _, p := range op.parameters {
			if !p.isStringType() {
				defineErr = true
			}
		}
	}

	if defineErr {
		fmt.Fprintf(f, "var err error\n")
		useFmt = true //use fmt package to write error
	}

	//write parameter extracting
	inputArgs := []string{}
	inputArgDefs := []string{}     //input argument definitions
	inputArgLongDefs := []string{} //input argument definitions with variables
	paramPrefix := ""

	if len(op.parameters) >= 2 {
		//write data structure to hold parameters
		paramStruct := op.id + "Params"
		fmt.Fprintf(f, "var params %s\n", paramStruct)
		inputArgs = append(inputArgs, "&params")
		inputArgDefs = append(inputArgDefs, "*"+paramStruct)
		inputArgLongDefs = append(inputArgLongDefs, fmt.Sprintf("params *%s.%s", _pkgName, paramStruct))
		paramPrefix = "params."
	} else {
		for _, p := range op.parameters {
			inputArgs = append(inputArgs, p.getName(false))
			inputArgDefs = append(inputArgDefs, p.getTypeDefinition())
			inputArgLongDefs = append(inputArgLongDefs, p.getDefinition(false))
		}
	}

	for _, p := range op.parameters {
		fmt.Fprintf(f, "%s\n", p.writeExtractParam(paramPrefix))
	}

	//write request decoding
	if op.requestBody != nil {
		fmt.Fprintf(f, "\n// decode request body\n")
		fmt.Fprintf(f, "contentLength, content := ctx.RequestBody()\n")
		bodyIsMapOrArray := op.requestBody.content.isArray || op.requestBody.content.isMap
		bodyModelName = op.requestBody.content.writeGoType("models.")
		if op.requestBody.required {
			if !bodyIsMapOrArray {
				fmt.Fprintf(f, "body := new(%s)\n", bodyModelName)
			}
			fmt.Fprintf(f, "if err = sbi.Decode(contentLength, content, body); err != nil {\n")
			fmt.Fprintf(f, "ctx.WriteResponse(400, models.CreateProblemDetails(400, fmt.Sprintf(\"Fail to decode request body: %%+v\", err)), nil)\nreturn\n")
			fmt.Fprintf(f, "}\n")
		} else {
			useIo = true
			if !bodyIsMapOrArray {
				fmt.Fprintf(f, "var body *%s\n", bodyModelName)
			} else {
				fmt.Fprintf(f, "var body %s\n", bodyModelName)
			}

			if !bodyIsMapOrArray {
				fmt.Fprintf(f, "body = new(%s)\n", bodyModelName)
			}

			fmt.Fprintf(f, "if err = sbi.Decode(contentLength, content, body); err !=nil && err != io.EOF {\n")
			fmt.Fprintf(f, "ctx.WriteResponse(400, models.CreateProblemDetails(400,fmt.Sprintf(\"Fail to decode request body: %%+v\", err)), nil)\nreturn\n")
			fmt.Fprintf(f, "}")

			if !bodyIsMapOrArray {
				fmt.Fprintf(f, "else if err == io.EOF {\nbody = nil\n}")
			}
		}

		inputArgs = append(inputArgs, "body")
		inputArgDefs = append(inputArgDefs, "*"+bodyModelName)
		inputArgLongDefs = append(inputArgLongDefs, "body *"+bodyModelName)
	}
	//write calling application handler
	outputs := []string{}
	outputDefs := []string{}     //output definitions
	outputLongDefs := []string{} //output definitions with variables
	checkOutputs := []string{}
	headerComments := ""

	if op.successModel != nil {
		headers := "nil"
		if len(op.sHeaders) > 0 {
			log.Warnf("Operator %s with a success response has headers: %v", op.id, op.sHeaders)
			outputs = append(outputs, "headers")
			outputDefs = append(outputDefs, "map[string]string")
			outputLongDefs = append(outputLongDefs, "headers map[string]string")
			headers = "headers"
			headerComments = "// headers: " + strings.Join(op.sHeaders, ", ") + "\n"
		}

		successModelName = op.successModel.writeGoType("models.")
		outputs = append(outputs, "rsp")
		outputDefs = append(outputDefs, "*"+successModelName)
		outputLongDefs = append(outputLongDefs, "rsp *"+successModelName)
		tmp := fmt.Sprintf("\n// check for success response\n if rsp != nil {\nctx.WriteResponse(%s, rsp, %s)\nreturn\n}\n", op.successCode, headers)
		checkOutputs = append(checkOutputs, tmp)
	}

	if len(op.errorCodes) > 0 && op.errorModel != nil {
		headers := "nil"
		if len(op.eHeaders) > 0 {
			log.Warnf("Operator %s with error response has headers: %v", op.id, op.eHeaders)
			outputs = append(outputs, "headers")
			outputDefs = append(outputDefs, "map[string]string")
			outputLongDefs = append(outputLongDefs, "headers map[string]string")
			headers = "headers"
			headerComments = "// headers: " + strings.Join(op.eHeaders, ", ") + "\n"
		}

		errorModelName = op.errorModel.writeGoType("models.")
		outputs = append(outputs, "ersp")
		outputDefs = append(outputDefs, "*"+errorModelName)
		outputLongDefs = append(outputLongDefs, "ersp *"+errorModelName)

		var errCode string
		if len(op.errorCodes) == 1 {
			errCode = op.errorCodes[0]
		} else {
			errCode = fmt.Sprintf("models.StatusFrom%s(ersp)", op.errorModel.goType)
		}
		tmp := fmt.Sprintf("\n// check for defined error\n if ersp != nil {\nctx.WriteResponse(%s, ersp, %s)\nreturn\n}\n", errCode, headers)
		checkOutputs = append(checkOutputs, tmp)
	}

	if len(op.problemCodes) > 0 {
		outputs = append(outputs, "prob")
		outputDefs = append(outputDefs, "*models.ProblemDetails")
		outputLongDefs = append(outputLongDefs, "prob *models.ProblemDetails")
		tmp := fmt.Sprintf("\n // check for problem\n if prob != nil {\nctx.WriteResponse(prob.Status, prob, nil)\nreturn\n}\n")
		checkOutputs = append(checkOutputs, tmp)
	}

	if len(op.emptySuccessCodes) > 0 {
		tmp := fmt.Sprintf("\n// success\nctx.WriteResponse(%s, nil, nil)\n", op.emptySuccessCodes[0])
		checkOutputs = append(checkOutputs, tmp)
	}

	fmt.Fprintf(f, "\n// call application handler\n")
	if len(outputs) > 0 {
		fmt.Fprintf(f, "%s := prod.Handle%s(%s)\n", strings.Join(outputs, ","), op.id, strings.Join(inputArgs, ","))
	} else {
		fmt.Fprintf(f, "prod.Handle%s(%s)\n", op.id, strings.Join(inputArgs, ","))
	}

	methodImpl = fmt.Sprintf("func (p *Producer) Handle%s(%s)(%s){\n%s\treturn\n}", op.id, strings.Join(inputArgLongDefs, ","), strings.Join(outputLongDefs, ","), headerComments)
	methodInf = fmt.Sprintf("Handle%s(%s)(%s)\n", op.id, strings.Join(inputArgDefs, ","), strings.Join(outputDefs, ","))

	fmt.Fprintf(f, strings.Join(checkOutputs, ""))
	fmt.Fprintf(f, "\n}\n")
	return
}

func writeConsumerApi(f *os.File, op *Operation) {
	fmt.Fprintf(f, "//Summary: %s\n", op.summary)
	fmt.Fprintf(f, "//Description: %s\n", op.desc)
	fmt.Fprintf(f, "//Path: %s\n", op.path)
	//fmt.Fprintf(f, "//Path template: %s\n", op.pathTmpl)
	fmt.Fprintf(f, "//Path Params: %s\n", strings.Join(op.pathParams, ", "))
	if len(op.sHeaders) > 0 {
		fmt.Fprintf(f, "//Response headers: %s\n", strings.Join(op.sHeaders, ", "))
	} else if len(op.eHeaders) > 0 {
		fmt.Fprintf(f, "//Response headers: %s\n", strings.Join(op.eHeaders, ", "))
	}

	inputArgs := []string{"cli sbi.ConsumerClient"}
	paramPrefix := ""

	if len(op.parameters) >= 2 {
		//write data structure to hold parameters
		paramStruct := fmt.Sprintf("%sParams", op.id)
		fmt.Fprintf(f, "type %s struct {\n", paramStruct)
		for _, p := range op.parameters {
			fmt.Fprintf(f, "%s\n", p.getDefinition(true))
		}
		fmt.Fprintf(f, "}\n")
		inputArgs = append(inputArgs, fmt.Sprintf("params %s", paramStruct))
		paramPrefix = "params."
	} else {
		for _, p := range op.parameters {
			inputArgs = append(inputArgs, p.getDefinition(false))
		}
	}

	if op.requestBody != nil {
		body := op.requestBody.content.writeGoType("models.")
		inputArgs = append(inputArgs, fmt.Sprintf("body *%s", body))
	}

	var successModelName, errorModelName string
	outputArgs := []string{}
	if len(op.sHeaders) > 0 {
		outputArgs = append(outputArgs, "headers map[string]string")
	} else if len(op.eHeaders) > 0 {
		outputArgs = append(outputArgs, "headers map[string]string")
	}

	if op.successModel != nil {
		successModelName = op.successModel.writeGoType("models.")
		outputArgs = append(outputArgs, fmt.Sprintf("rsp *%s", successModelName))
	}

	if len(op.errorCodes) > 0 && op.errorModel != nil {
		errorModelName = op.errorModel.writeGoType("models.")
		outputArgs = append(outputArgs, fmt.Sprintf("ersp *%s", errorModelName))
	}

	outputArgs = append(outputArgs, "err error")
	//write function definition
	fmt.Fprintf(f, "func %s(%s) (%s) {\n", op.id, strings.Join(inputArgs, ","), strings.Join(outputArgs, ","))

	fmt.Fprintf(f, "\n")

	//write checking required params
	paramChecks := []string{}
	for _, p := range op.parameters {
		if check := p.writeParamCheck(paramPrefix); len(check) > 0 {
			paramChecks = append(paramChecks, check)
		}
	}
	if len(paramChecks) > 0 {
		fmt.Fprintf(f, strings.Join(paramChecks, "\n"))
		fmt.Fprintf(f, "\n")
	}

	//write check body required
	if op.requestBody != nil {
		if op.requestBody.required {
			fmt.Fprintf(f, "if body == nil {\n err = fmt.Errorf(\"body is required\")\nreturn\n}\n")
		}
	}

	//write path
	if len(op.pathParams) > 0 {
		pathParams := []string{}
		for _, pName := range op.pathParams {
			p, _ := op.parameters[pName]
			pathParams = append(pathParams, p.stringConvertFn(paramPrefix))
		}
		fmt.Fprintf(f, "\npath:= fmt.Sprintf(\"%%s%s\",PATH_ROOT, %s)\n", op.pathTmpl, strings.Join(pathParams, ", "))
	} else {
		fmt.Fprintf(f, "\npath:= fmt.Sprintf(\"%%s%s\",PATH_ROOT)\n", op.pathTmpl)
	}

	//write send request
	if op.requestBody != nil {
		fmt.Fprintf(f, "request := sbi.NewRequest(path, %s, body)\n", op.writeMethod())
	} else {
		fmt.Fprintf(f, "request := sbi.NewRequest(path, %s, nil)\n", op.writeMethod())
	}

	//write adding querry/header params to request
	paramAdders := []string{}
	for _, p := range op.parameters {
		if adder := p.writeParamAdd(paramPrefix); len(adder) > 0 {
			paramAdders = append(paramAdders, adder)
		}
	}
	if len(paramAdders) > 0 {
		fmt.Fprintf(f, strings.Join(paramAdders, "\n"))
		fmt.Fprintf(f, "\n")
	}

	fmt.Fprintf(f, "var response *sbi.Response\n")
	fmt.Fprintf(f, "if response, err = cli.Send(request); err !=nil {\n return\n}\n\n")

	fmt.Fprintf(f, "defer response.CloseBody()\n\n")

	//write check for response
	fmt.Fprintf(f, "switch response.GetCode() {\n")
	if op.successModel != nil {
		fmt.Fprintf(f, "case %s:\n", op.successCode)
		if len(op.sHeaders) > 0 {
			fmt.Fprintf(f, "headers = response.GetHeaders()\n")
		}

		fmt.Fprintf(f, "rsp = new(%s)\n if err = response.DecodeBody(rsp); err != nil {err = fmt.Errorf(\"Fail to decode %s: %%+v\", err)}\n", successModelName, op.successModel.writeGoType(""))
	}
	if len(op.emptySuccessCodes) > 0 {
		fmt.Fprintf(f, "case %s:\nreturn\n", strings.Join(op.emptySuccessCodes, ","))
	}

	if len(op.errorCodes) > 0 && op.errorModel != nil {
		fmt.Fprintf(f, "case %s:\n", strings.Join(op.errorCodes, ","))
		fmt.Fprintf(f, "ersp = new(%s)\n if err = response.DecodeBody(ersp); err != nil {err = fmt.Errorf(\"Fail to decode %s: %%+v\", err)}\n", errorModelName, op.errorModel.writeGoType(""))
		if len(op.eHeaders) > 0 {
			fmt.Fprintf(f, "headers = response.GetHeaders()\n")
		}
	}

	if len(op.problemCodes) > 0 {
		fmt.Fprintf(f, "case %s:\n", strings.Join(op.problemCodes, ","))
		fmt.Fprintf(f, "prob := new(models.ProblemDetails)\n if err = response.DecodeBody(prob); err == nil {\nerr=sbi.ErrorFromProblemDetails(prob)\n}else {err = fmt.Errorf(\"Fail to decode ProblemDetails: %%+v\", err)\n}\n")
	}

	fmt.Fprintf(f, "default:\nerr=fmt.Errorf(\"%%d, %%s\",response.GetCode(), response.GetStatus())\n}\n")
	fmt.Fprintf(f, "return\n}\n")
}

func readPaths(paths *orderedmap.Map[string, *v3.PathItem]) map[string]Operation {
	operations := make(map[string]Operation)
	for pathPair := paths.First(); pathPair != nil; pathPair = pathPair.Next() {
		// get the name of the schema
		pathName := pathPair.Key()

		// get the schema object from the map
		pathItem := pathPair.Value()
		for _, op := range readPathItem(pathName, pathItem) {
			operations[op.id] = *op
		}
	}
	return operations
}

func createOpId(path string) string {
	parts := strings.FieldsFunc(path, func(c rune) bool {
		return c == '/' || c == '-' || c == '_' || c == ' '
	})
	idParts := []string{}
	for _, p := range parts {
		if len(p) > 0 && p[0] != '{' {
			idParts = append(idParts, strings.Title(p))
		}
	}
	tmp := strings.Join(idParts, "")
	if tmp[0] == '3' {
		return "Three" + strings.Title(tmp[1:])
	}

	if tmp[0] == '5' {
		return "Five" + strings.Title(tmp[1:])
	}
	return tmp
}

func getRepresentativeContentModel(content *orderedmap.Map[string, *v3.MediaType]) *DataModel {
	if content == nil {
		return nil
	}
	var selectedModel *DataModel = nil
	for pair := content.First(); pair != nil; pair = pair.Next() {
		if m := analyzeSchema("", pair.Value().Schema); m != nil {
			if len(m.goType) == 0 {
				//return the first model created from inline schema
				return m
			} else if !isPrimitive(m.goType) {
				if selectedModel == nil {
					selectedModel = m
				} else {
					if m.goType != "ProblemDetails" {
						selectedModel = m
					}
				}
			}
		}
	}
	return selectedModel
}
func addNewModel(id string, m *DataModel) *DataModel {
	newM := new(DataModel)
	*newM = *m
	newM.id = id
	newM.goType = id
	models[id] = *newM
	return newM
}

func readPathItem(path string, item *v3.PathItem) (list []*Operation) {
	opList := make(map[string]*v3.Operation)
	var opStr string
	if item.Get != nil {
		opStr = "GET"
		opList[opStr] = item.Get
	}
	if item.Put != nil {
		opStr = "PUT"
		opList[opStr] = item.Put
	}
	if item.Post != nil {
		opStr = "POST"
		opList[opStr] = item.Post
	}
	if item.Delete != nil {
		opStr = "DELETE"
		opList[opStr] = item.Delete
	}
	if item.Patch != nil {
		opStr = "PATCH"
		opList[opStr] = item.Patch
	}

	for method, op := range opList {
		if opModel := createOperation(path, method, item.Parameters, op); opModel != nil {
			list = append(list, opModel)
		}
	}
	return
}

func createOperation(path string, method string, params []*v3.Parameter, op *v3.Operation) *Operation {
	opModel := &Operation{
		method:     method,
		summary:    op.Summary,
		desc:       op.Description,
		parameters: make(map[string]Parameter),
	}
	//create operation id
	if len(op.OperationId) == 0 {
		opModel.id = createOpId(path + "/" + strings.Title(strings.ToLower(method)))
	} else {
		opModel.id = createOpId(op.OperationId)
	}

	//parse parameters
	parameters := make(map[string]*v3.Parameter)
	for _, p := range params {
		parameters[p.Name] = p
	}
	for _, p := range op.Parameters {
		parameters[p.Name] = p
	}

	pathParams := []string{}
	for id, p := range parameters {
		var m *DataModel
		if m = analyzeSchema("", p.Schema); m == nil {
			//try to get it from content
			if p.Content != nil {
				for pair := p.Content.First(); pair != nil; pair = pair.Next() {
					if m = analyzeSchema("", pair.Value().Schema); m != nil {
						break
					}
				}
			}
		}

		if len(m.goType) == 0 {
			log.Errorf("Parameter %s of operation %s has inline-object type which is not supported", id, opModel.id)
			return nil
		}

		tmpP := Parameter{
			id:   id,
			m:    *m,
			in:   p.In,
			desc: p.Description,
		}
		if p.Required != nil {
			tmpP.required = *p.Required
		}
		opModel.parameters[tmpP.id] = tmpP
		if tmpP.in == "path" {
			pathParams = append(pathParams, tmpP.id)
		}
	}
	var err error
	if opModel.path, opModel.pathTmpl, opModel.pathParams, err = buildRoute(path, pathParams); err != nil {
		log.Errorf("Fail to build route template for the operation %s: %+v", opModel.id, err)
		return nil
	}

	//get request body
	if body := op.RequestBody; body != nil {
		if selectedModel := getRepresentativeContentModel(body.Content); selectedModel != nil {
			if len(selectedModel.goType) == 0 {
				//add model to the global repo
				modelId := fmt.Sprintf("%sRequest", opModel.id)
				selectedModel = addNewModel(modelId, selectedModel)
			}
			opModel.requestBody = &RequestBody{
				desc:    body.Description,
				content: *selectedModel,
			}
			if body.Required != nil {
				opModel.requestBody.required = *body.Required
			}
		}
	}

	if body := opModel.requestBody; body != nil {
		log.Infof("OP %s has body '%s[%v]'", opModel.id, body.content.id, body.required)
	} else {
		log.Infof("Op: '%s' has no request body", opModel.id)
	}

	//get responses
	for pair := op.Responses.Codes.First(); pair != nil; pair = pair.Next() {
		code := pair.Key()
		response := pair.Value()
		headers := []string{}
		if h := response.Headers; h != nil {
			for hPair := h.First(); hPair != nil; hPair = hPair.Next() {
				headers = append(headers, hPair.Key())
			}
		}

		if code[0] == '1' {
			log.Warnf("respone with Information Http Code not processed")
			continue
		} else if code[0] == '3' {
			log.Warnf("respone with redirection Http Code not processed")
			continue
		} else if code[0] == '2' { //success response
			if selectedModel := getRepresentativeContentModel(response.Content); selectedModel != nil {
				if len(selectedModel.goType) == 0 {
					//assign model Id and goType then add to the model repo
					modelId := fmt.Sprintf("%sResponse", opModel.id)
					opModel.successModel = addNewModel(modelId, selectedModel)
				} else {
					opModel.successModel = selectedModel
				}
				opModel.successCode = code
			} else { //add success code with empty response
				opModel.emptySuccessCodes = append(opModel.emptySuccessCodes, code)
			}
			opModel.sHeaders = headers
		} else { //error responses
			if selectedModel := getRepresentativeContentModel(response.Content); selectedModel != nil {
				if len(selectedModel.goType) == 0 {
					//assign model Id and goType then add to the model repo
					modelId := fmt.Sprintf("%sErrorResponse", opModel.id)
					selectedModel = addNewModel(modelId, selectedModel)
				}

				if selectedModel.goType == "ProblemDetails" {
					opModel.problemCodes = append(opModel.problemCodes, code)
				} else {
					opModel.errorModel = selectedModel
					opModel.errorCodes = append(opModel.errorCodes, code)
				}
			}

			opModel.eHeaders = headers
		}
	}
	if opModel.successModel != nil {
		log.Infof("OP %s: success response:%s [%s]", opModel.id, opModel.successModel.goType, opModel.successCode)
	}
	log.Infof("OP %s: codes with problem details:%v", opModel.id, opModel.problemCodes)
	log.Infof("OP %s: codes with error response:%v", opModel.id, opModel.errorCodes)
	log.Infof("OP %s: codes with empty response:%v", opModel.id, opModel.emptySuccessCodes)

	return opModel
}

func createModels(schemas *orderedmap.Map[string, *base.SchemaProxy]) {
	// print the number of paths and schemas in the document
	models = make(map[string]DataModel)

	for schemaPairs := schemas.First(); schemaPairs != nil; schemaPairs = schemaPairs.Next() {
		// get the name of the schema
		schemaName := schemaPairs.Key()

		// get the schema object from the map
		schemaValue := schemaPairs.Value()

		// build the schema
		schema := schemaValue.Schema()

		if schema == nil {
			log.Errorf("EMPTY SCHEMA '%s'\n", schemaName)
		} else {
			analyzeSchema(schemaName, schemaValue)
		}
	}
}
func getSchemaIdFromRef(ref string) string {
	tokens := strings.Split(ref, "/")
	if l := len(tokens); l > 0 {
		return tokens[l-1]
	}
	return ""
}

func analyzeAllOf(id string, allOf []*base.SchemaProxy) *DataModel {
	out := &DataModel{
		id:         id,
		goType:     id,
		properties: make(map[string]Property),
	}

	for _, s := range allOf {
		if m := analyzeSchema("", s); m != nil {
			for pId, p := range m.properties {
				out.properties[pId] = p
			}
		}
	}
	return out
}

func analyzeOneOf(id string, oneOf []*base.SchemaProxy) *DataModel {
	return analyzeAnyOf(id, oneOf)
}
func analyzeAnyOf(id string, anyOf []*base.SchemaProxy) *DataModel {

	isArray := false
	goTypes := make(map[string]DataModel)
	var enum *Enum
	var subModels []DataModel
	for _, s := range anyOf {
		if m := analyzeSchema("", s); m != nil {
			if m.isArray {
				isArray = true
			}
			if len(m.goType) > 0 {
				goTypes[m.goType] = *m
			}
			if m.enum != nil {
				enum = m.enum
			}
			subModels = append(subModels, *m)
		}
	}
	if len(goTypes) == 0 {
		return nil
	} else if len(goTypes) > 1 {
		if len(id) == 0 {
			return nil
		}

		log.Infof("ANYOF %s has more than one types", id)
		m := &DataModel{
			id:         id,
			goType:     makeModelName(id),
			properties: make(map[string]Property),
		}
		for tStr, t := range goTypes {
			if len(t.id) > 0 {
				tStr = makeModelName(t.id)
			}
			m.properties[tStr] = Property{
				id:       tStr,
				m:        &t,
				required: false,
			}
		}
		return m
	} else {
		if len(subModels) == 1 {
			m := subModels[0]
			if len(m.id) == 0 {
				m.id = id
			}
			return &m
		} else {
			m := &DataModel{
				isArray: isArray,
				id:      id,
				enum:    enum,
			}
			for t := range goTypes {
				m.goType = t
				break
			}
			return m
		}
	}
	return nil
}

func analyzeSchema(id string, schemaP *base.SchemaProxy) *DataModel {
	if schemaP == nil {
		return nil
	}
	if len(id) == 0 {
		if schemaP.IsReference() {
			id = getSchemaIdFromRef(schemaP.GetReference())
		}
	}
	if len(id) > 0 {
		if m, ok := models[id]; ok {
			return &m
		}
	}
	schema := schemaP.Schema()

	if schema == nil {
		return nil
	}

	if len(schema.Type) > 1 {
		log.Errorf("Schema with multiple types not supported")
		return nil
	}
	var m *DataModel
	if len(schema.Type) == 0 {
		if len(id) == 0 {
			if keyNode := schemaP.GetSchemaKeyNode(); keyNode != nil {
				id = keyNode.Value
			}
		}

		if len(schema.AllOf) > 0 {
			m = analyzeAllOf(id, schema.AllOf)
		} else if len(schema.AnyOf) > 0 {
			m = analyzeAnyOf(id, schema.AnyOf)
		} else if len(schema.OneOf) > 0 {
			m = analyzeOneOf(id, schema.OneOf)
		} else {
			//log.Warnf("UNTYPE schema %s", id)
			return nil
		}
	} else {
		m = &DataModel{
			id:         id,
			schema:     schema,
			properties: make(map[string]Property),
		}
		switch schema.Type[0] {
		case "object":
			m.goType = makeModelName(id)
			if schema.AdditionalProperties != nil {
				m.isMap = true
				if schema.AdditionalProperties.IsA() {
					if refModel := analyzeSchema("", schema.AdditionalProperties.A); refModel != nil {
						m.goType = refModel.goType
						m.isExternal = refModel.isExternal
					} else {
						m.goType = "UNKNOWN"
					}
				} else {
					m.goType = "bool"
				}
				//log.Infof("ADDPROP:map[string]%s", m.goType)
			} else {
				for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
					propName := pair.Key()
					propSchemaProxy := pair.Value()
					propSchema := propSchemaProxy.Schema()
					if propSchema == nil {
						log.Warnf("%s has empty attribute %s", id, propName)
					} else {
						p := Property{
							id:       propName,
							required: inList(propName, schema.Required),
						}

						if refModel := analyzeSchema("", propSchemaProxy); refModel != nil {
							p.m = refModel
						}
						m.properties[p.id] = p
					}
				}
				var extra *DataModel
				if l := len(schema.AllOf); l > 0 {
					extra = analyzeAllOf("", schema.AllOf)
				} else if l = len(schema.AnyOf); l > 0 {
					extra = analyzeAnyOf("", schema.AnyOf)
				} else if l = len(schema.OneOf); l > 0 {
					extra = analyzeOneOf("", schema.OneOf)
				}
				if extra != nil {
					for key, value := range extra.properties {
						m.properties[key] = value
					}
				}
			}
		case "string":
			if schema.Format == "binary" {
				m.goType = "byte"
				m.isArray = true
			} else {
				m.goType = "string"
			}
		case "integer":
			switch schema.Format {
			case "int16":
				m.goType = "int16"
			case "int32":
				m.goType = "int32"
			case "int64":
				m.goType = "int64"
			default:
				m.goType = "int"
			}
		case "number":
			switch schema.Format {
			case "double":
				m.goType = "float64"
			default:
				m.goType = "float64"
			}
		case "boolean":
			m.goType = "bool"

		case "array":
			m.isArray = true
			if schema.Items != nil && schema.Items.IsA() {
				if itemModel := analyzeSchema("", schema.Items.A); itemModel != nil {
					m.goType = itemModel.goType
					m.isExternal = itemModel.isExternal
				} else {
					m.goType = "[]Unknown"
				}
			} else {
				m.goType = "bool"
			}
		case "external":
			if len(id) == 0 {
				log.Errorf("external schema without identity")
				return nil
			}
			m.isExternal = true
			m.goType = id
		default:
			log.Errorf("Not supported type: %s", schema.Type[0])
			return nil
		}
	}

	if len(schema.Enum) > 0 {
		m.enum = &Enum{
			enumType: m.goType,
		}
		for _, node := range schema.Enum {
			m.enum.values = append(m.enum.values, node.Value)
		}
		if len(id) == 0 {
			if keyNode := schemaP.GetSchemaKeyNode(); keyNode != nil {
				id = keyNode.Value
			}
		}
		m.id = makeModelName(id)
	}

	if m != nil && len(m.id) > 0 {
		models[m.id] = *m
	}
	return m
}

var primitives []string = []string{"bool", "int", "int16", "int32", "int64", "float32", "float64", "string", "byte"}

func isPrimitive(t string) bool {
	return inList(t, primitives)
}

func indexInList(item string, list []string) int {
	for i, s := range list {
		if item == s {
			return i
		}
	}
	return -1
}

func inList(item string, list []string) bool {
	return indexInList(item, list) != -1
}

func makeModelName(s string) string {
	if len(s) == 0 {
		return ""
	}

	parts := strings.FieldsFunc(s, func(c rune) bool {
		return c == ' ' || c == '-' || c == '_'
	})
	out := ""
	for _, p := range parts {
		out = out + strings.Title(p)
	}

	if out[0] == '5' {
		out = "Five" + strings.Title(out[1:])
	}

	if out[0] == '3' {
		out = "Three" + strings.Title(out[1:])
	}

	return out
}
