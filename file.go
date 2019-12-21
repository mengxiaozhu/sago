package sago

import (
	"encoding/xml"
	"errors"
	"github.com/mengxiaozhu/linkerror"
)

type SQLContent struct {
	Name string `xml:"name,attr"`
	Args string `xml:"args,attr"`
	SQL  string `xml:",chardata"`
}

type File struct {
	XMLName  xml.Name     `xml:"sago"`
	Package  string       `xml:"package"`
	Type     string       `xml:"type"`
	Table    string       `xml:"table"`
	Selects  []SQLContent `xml:"select"`
	Executes []SQLContent `xml:"execute"`
	Inserts  []SQLContent `xml:"insert"`
}

func (f File) Name() string {
	if f.Package == "" {
		return f.Type
	}
	return f.Package + "." + f.Type
}

type Fn struct {
	Name string
	Type string
	SQL  string
	Args []string
}

type SQLSet struct {
	Package   string
	Type      string
	Table     string
	Functions map[string]*Fn
}

var FileConflict = errors.New("file conflict")

func combineFiles(r1, r2 *File) (r *File, err *linkerror.Error) {
	r = &File{Table: r1.Table}
	if r2.Table != r1.Table {
		return nil, linkerror.New(FileConflict, r.Package+"."+r.Type+" conflict "+r2.Table+"!="+r1.Table)
	}
	r.Executes = combineSQLContent(r1.Executes, r2.Executes)
	r.Selects = combineSQLContent(r1.Selects, r2.Selects)
	r.Inserts = combineSQLContent(r1.Inserts, r2.Inserts)
	return
}

func combineSQLContent(c1, c2 []SQLContent) []SQLContent {
	sqlContents := make([]SQLContent, 0, len(c1)+len(c2))
	sqlContents = append(sqlContents, c1...)
	sqlContents = append(sqlContents, c2...)
	return sqlContents
}
