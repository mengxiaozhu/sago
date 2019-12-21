package sago

import (
	"encoding/xml"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
	"strings"
)

const (
	MethodNameArg = "arg"
	MethodInArg   = "in"
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
func inFunc(ctx *FnCtx) TemplateFunc {
	return func(args interface{}) (string, error) {
		v := reflect.ValueOf(args)
		length := v.Len()
		for i := 0; i < length; i++ {
			ctx.Args = append(ctx.Args, v.Index(i).Interface())
		}
		return "in (" + strings.Repeat("?,", length-1) + "?)", nil
	}
}

func parseXML(path string) (f *File, err error) {
	xmlData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f = &File{
		Selects:  []SQLContent{},
		Executes: []SQLContent{},
		Inserts:  []SQLContent{},
	}
	err = xml.Unmarshal(xmlData, f)
	return
}

func parseYAML(path string) (f *File, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f = &File{
		Selects:  []SQLContent{},
		Executes: []SQLContent{},
		Inserts:  []SQLContent{},
	}
	err = yaml.Unmarshal(data, f)
	return
}
