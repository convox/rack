package cgo

import (
	"fmt"
	"go/ast"
	"regexp"
)

var cgoImports = `// #include <stdio.h>
// #include <stdlib.h>
// #include <string.h>
// #include <stdio.h>
import "C"

import "github.com/aws/aws-sdk-go/aws/credentials"
`

const cgoExportsTemplate = `

//export Lambda
func Lambda(functionName *C.char,
	logLevel *C.char,
	requestJSON *C.char,
	accessKeyID *C.char,
	secretKey *C.char,
	token *C.char,
	exitCode *C.int,
	responseContentTypeBuffer *C.char,
	responseContentTypeLen int,
	responseBuffer *C.char,
	responseBufferContentLen int) int {

	inputFunction := C.GoString(functionName)
	golangLogLevel := C.GoString(logLevel)
	inputRequest := C.GoString(requestJSON)
	awsCreds := credentials.NewStaticCredentials(C.GoString(accessKeyID),
		C.GoString(secretKey),
		C.GoString(token))

	spartaResp, spartaRespHeaders, responseErr := %s.LambdaHandler(inputFunction, golangLogLevel, inputRequest, awsCreds)
	lambdaExitCode := 0
	var pyResponseBufer []byte
	if nil != responseErr {
		lambdaExitCode = 1
		pyResponseBufer = []byte(responseErr.Error())
	} else {
		pyResponseBufer = spartaResp
	}

	// Copy content type
	contentTypeHeader := spartaRespHeaders.Get("Content-Type")

	// If there is no header, assume it's json
	if "" == contentTypeHeader {
		contentTypeHeader = "application/json"
	}
	if "" != contentTypeHeader {
		responseContentTypeBytes := C.CBytes([]byte(contentTypeHeader))
		defer C.free(responseContentTypeBytes)
		copyContentTypeBytesLen := len(contentTypeHeader)
		if (copyContentTypeBytesLen > responseContentTypeLen) {
			copyContentTypeBytesLen = responseContentTypeLen
		}
		C.memcpy(unsafe.Pointer(responseContentTypeBuffer),
			unsafe.Pointer(responseContentTypeBytes),
			C.size_t(copyContentTypeBytesLen))
	}

	// Copy response body
	copyBytesLen := len(pyResponseBufer)
	if copyBytesLen > responseBufferContentLen {
		copyBytesLen = responseBufferContentLen
	}
	responseBytes := C.CBytes(pyResponseBufer)
	defer C.free(responseBytes)
	C.memcpy(unsafe.Pointer(responseBuffer),
		unsafe.Pointer(responseBytes),
		C.size_t(copyBytesLen))
	*exitCode = C.int(lambdaExitCode)
	return copyBytesLen
}

func main() {
	// NOP
}
`

var packageRegexp = regexp.MustCompile("(?m)^package.*[\r\n]{1,2}")

// First thing we need to do is change the main() function
// to be an init() function
type mainRewriteVisitor struct {
}

func (v *mainRewriteVisitor) Visit(node ast.Node) (w ast.Visitor) {
	switch t := node.(type) {
	case *ast.FuncDecl:
		if t.Name.Name == "main" {
			t.Name = ast.NewIdent("init")
		}
	}
	return v
}

func visitors() []ast.Visitor {
	return []ast.Visitor{
		&mainRewriteVisitor{},
	}
}

type transformer func(inputSource string, cgoPackageAlias string) (string, error)

func cgoImportsTransformer(inputSource string, cgoPackageAlias string) (string, error) {
	matchIndex := packageRegexp.FindStringIndex(inputSource)
	if nil == matchIndex {
		return "", fmt.Errorf("Failed to find package statement")
	}
	// Great, append the cgo header
	return fmt.Sprintf("%s%s%s",
		inputSource[0:matchIndex[1]],
		cgoImports,
		inputSource[matchIndex[1]:]), nil
}

func cgoExportsTransformer(inputSource string, cgoPackageAlias string) (string, error) {
	return fmt.Sprintf("%s\n%s",
			inputSource,
			fmt.Sprintf(cgoExportsTemplate, cgoPackageAlias)),
		nil
}

func transformers() []transformer {
	return []transformer{
		cgoImportsTransformer,
		cgoExportsTransformer,
	}
}
