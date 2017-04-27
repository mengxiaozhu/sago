package sago

import (
	"encoding/xml"
	"io/ioutil"
	"reflect"
)

const (
	MethodName_Arg = "arg"
)

var emptyReflectValue = reflect.Value{}

func empty(v interface{}) (string, error) {
	return "", nil
}

type FnCtx struct {
	Args []interface{}
}

func argFunc(ctx *FnCtx) TemplateFunc {
	return func(args interface{}) (string, error) {
		ctx.Args = append(ctx.Args, args)
		return "?", nil
	}
}

func parseXML(path string) (root *XMLRoot, err error) {
	xmlData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	root = &XMLRoot{
		Selects:  []SQLContent{},
		Executes: []SQLContent{},
		Inserts:  []SQLContent{},
	}
	err = xml.Unmarshal(xmlData, root)
	return

}
