package sago

type errorSago int

const (
	Dir errorSago = iota // dir error
	Xml                  // unmarshal xml error
	YAML
	NoDBField
	NoCacheInManager
	BadCacheField
	BadSQLTemplate
	WrongTypeToMap
	XMLMappedWrong
)

var errorSagoError = [...]string{
	Dir:              "read dir wrong",
	Xml:              "unmarshal xml error",
	YAML:             "unmarshal yaml error",
	NoDBField:        "no DB field in struct",
	NoCacheInManager: "manager's Cache field is nil",
	BadCacheField:    "Cache field of this struct is wrong type for sago. the field type must be pointer of the struct type",
	BadSQLTemplate:   "bad sql",
	WrongTypeToMap:   "map object must be pointer of struct",
	XMLMappedWrong:   "XML mapped wrong",
}

func (m errorSago) Error() string {
	return errorSagoError[m]
}
