package net_rpc

import (
	"reflect"
	"runtime"
	"unicode"
	"unicode/utf8"

	"github.com/hezhis/go_log"
)

type methodType struct {
	method  reflect.Method
	ArgType reflect.Type
}

type functionType struct {
	fn      reflect.Value
	ArgType reflect.Type
}

type service struct {
	name     string                   // name of service
	rcvr     reflect.Value            // receiver of methods for the service
	typ      reflect.Type             // type of the receiver
	method   map[string]*methodType   // registered methods
	function map[string]*functionType // registered functions
}

func (s *service) call(mtype *methodType, argv reflect.Value) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			buf = buf[:n]

			logger.Error("[service internal error]: %v, method: %s, argv: %+v, stack: %s",
				r, mtype.method.Name, argv.Interface(), buf)
		}
	}()

	function := mtype.method.Func
	// Invoke the method, providing a new value for the reply.
	function.Call([]reflect.Value{s.rcvr, argv})

	return
}

func (s *service) callForFunction(ft *functionType, argv reflect.Value) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			buf = buf[:n]

			logger.Error("[service internal error]: %v, function: %s, argv: %+v, stack: %s",
				r, runtime.FuncForPC(ft.fn.Pointer()), argv.Interface(), buf)
		}
	}()

	ft.fn.Call([]reflect.Value{argv})
}

func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}
