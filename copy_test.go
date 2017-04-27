package sago

import (
	"code.aliyun.com/mougew/mengxiaozhu/models/mxz"
	"log"
	"reflect"
	"testing"
)

func TestCopy(t *testing.T) {
	list := []*mxz.MxzOtherUser{
		{
			Id:     1,
			Module: "asd",
		},
	}
	fuck := Copy(reflect.ValueOf(list)).Interface().([]*mxz.MxzOtherUser)
	log.Println(fuck[0])
	fuck[0].Module = "a;slkdal;sdjka"
	log.Println(fuck[0])
	log.Println(list[0])

}
