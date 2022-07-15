package net_rpc

import (
	"errors"
	"fmt"
	logger "github.com/hezhis/go_log"
	"github.com/hezhis/net_rpc/protocol"
	"github.com/hezhis/net_rpc/share"
	"reflect"
	"runtime"
	"strings"
)

var (
	FuncKindError = errors.New("function must be func or bound method")
)

type Server struct {
	serviceMap map[string]*service
}

func NewServer(opts ...OptionFn) *Server {
	s := &Server{
		serviceMap: make(map[string]*service),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Server) Register(rcvr interface{}) error {
	service := new(service)
	service.typ = reflect.TypeOf(rcvr)
	service.rcvr = reflect.ValueOf(rcvr)
	sName := reflect.Indirect(service.rcvr).Type().Name() // Type

	if sName == "" {
		errorStr := "net_rpc.Register: no service name for type " + service.typ.String()
		logger.Error(errorStr)
		return errors.New(errorStr)
	}
	if !isExported(sName) {
		errorStr := "net_rpc.Register: type " + sName + " is not exported"
		logger.Error(errorStr)
		return errors.New(errorStr)
	}
	service.name = sName

	// Install the methods
	service.method = suitableMethods(service.typ, true)

	if len(service.method) == 0 {
		var errorStr string

		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PtrTo(service.typ), false)
		if len(method) != 0 {
			errorStr = "net_rpc.Register: type " + sName + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			errorStr = "net_rpc.Register: type " + sName + " has no exported methods of suitable type"
		}
		logger.Error(errorStr)
		return errors.New(errorStr)
	}
	s.serviceMap[service.name] = service
	return nil
}

// suitableMethods returns suitable Rpc methods of typ, it will report
// error using log if reportErr is true.
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs four ins: receiver, *args.
		if mtype.NumIn() != 2 {
			if reportErr {
				logger.Debug("method %s has wrong number of ins: numIn:%d", mname, mtype.NumIn())
			}
			continue
		}

		// first arg need not be a pointer.
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				logger.Info(mname, " parameter type not exported: ", argType)
			}
			continue
		}

		methods[mname] = &methodType{method: method, ArgType: argType}

		// init pool for reflect.Type of args and reply
		reflectTypePools.Init(argType)
	}
	return methods
}

// RegisterFunction publishes a function that satisfy the following conditions:
// The client accesses function using a string of the form "servicePath.Method".
func (s *Server) RegisterFunction(servicePath string, fn interface{}) error {
	return s.registerFunction(servicePath, fn)
}

func (s *Server) registerFunction(servicePath string, fn interface{}) error {
	ss := s.serviceMap[servicePath]
	if ss == nil {
		ss = new(service)
		ss.name = servicePath
		ss.function = make(map[string]*functionType)
	}
	f, ok := fn.(reflect.Value)
	if !ok {
		f = reflect.ValueOf(fn)
	}
	if f.Kind() != reflect.Func {
		return FuncKindError
	}

	fName := runtime.FuncForPC(reflect.Indirect(f).Pointer()).Name()
	if fName != "" {
		i := strings.LastIndex(fName, ".")
		if i >= 0 {
			fName = fName[i+1:]
		}
	}
	if fName == "" {
		errorStr := "registerFunction: no func name for type " + f.Type().String()
		logger.Error(errorStr)
		return errors.New(errorStr)
	}

	t := f.Type()
	if t.NumIn() != 1 {
		return fmt.Errorf("rpcx.registerFunction: has wrong number of ins: %s", f.Type().String())
	}

	argType := t.In(0)
	if !isExportedOrBuiltinType(argType) {
		return fmt.Errorf("function %s parameter type not exported: %v", f.Type().String(), argType)
	}

	// Install the methods
	ss.function[fName] = &functionType{fn: f, ArgType: argType}
	s.serviceMap[servicePath] = ss

	// init pool for reflect.Type of args and reply
	reflectTypePools.Init(argType)

	return nil
}

func (s *Server) DoCall(req *protocol.Message) error {
	serviceName := req.ServicePath
	methodName := req.ServiceMethod

	service := s.serviceMap[serviceName]
	if service == nil {
		return errors.New("remotecall: can't find service " + serviceName)
	}
	mType := service.method[methodName]
	if mType == nil {
		if service.function[methodName] != nil { // check raw functions
			return s.callFunc(req)
		}
		return errors.New("remotecall: can't find method " + methodName)
	}

	// get a argv object from object pool
	argv := reflectTypePools.Get(mType.ArgType)

	codec := share.Codecs[req.SerializeType()]
	if codec == nil {
		return fmt.Errorf("can not find codec for %d", req.SerializeType())
	}

	err := codec.Decode(req.Payload, argv)
	if err != nil {
		return err
	}

	if mType.ArgType.Kind() != reflect.Ptr {
		service.call(mType, reflect.ValueOf(argv).Elem())
	} else {
		service.call(mType, reflect.ValueOf(argv))
	}

	// return argc to object pool
	reflectTypePools.Put(mType.ArgType, argv)

	return nil
}

func (s *Server) callFunc(req *protocol.Message) error {
	serviceName := req.ServicePath
	methodName := req.ServiceMethod

	service := s.serviceMap[serviceName]
	if service == nil {
		return errors.New("remotecall: can't find service  for func raw function")
	}
	mType := service.function[methodName]
	if mType == nil {
		return errors.New("remotecall: can't find method " + methodName)
	}

	argv := reflectTypePools.Get(mType.ArgType)

	codec := share.Codecs[req.SerializeType()]
	if codec == nil {
		return fmt.Errorf("can not find codec for %d", req.SerializeType())
	}

	err := codec.Decode(req.Payload, argv)
	if err != nil {
		return err
	}

	if mType.ArgType.Kind() != reflect.Ptr {
		service.callForFunction(mType, reflect.ValueOf(argv).Elem())
	} else {
		service.callForFunction(mType, reflect.ValueOf(argv))
	}

	reflectTypePools.Put(mType.ArgType, argv)

	return nil
}
