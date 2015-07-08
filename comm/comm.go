package comm

import (
	"encoding/json"
	"github.com/golang/glog"
	"reflect"
	"time"
)

func ToJsonFunc(arg interface{}) string {
	typ := reflect.TypeOf(arg)
	val := reflect.ValueOf(arg)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}
	if reflect.DeepEqual(val.Interface(), reflect.Zero(typ).Interface()) {
		switch typ.Kind() {
		case reflect.String:
			return ""
		case reflect.Struct:
			return "{}"
		case reflect.Slice, reflect.Map, reflect.Array:
			return "[]"
		}
	}
	j, _ := json.Marshal(arg)
	return string(j)
}

func CurrYear() int {
	return time.Now().Year()
}

func CurrMonth() int {
	return int(time.Now().Month())
}

func Substr(str string, start, length int) string {
	runes := []rune(str)
	rl := len(runes)
	end := 0

	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}

	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}

	return string(runes[start:end])
}

func ErrLog(err error) error {
	if err != nil {
		glog.Errorln(err)
	}
	return err
}

func FatalLog(err error) error {
	if err != nil {
		glog.Fatalln(err)
	}
	return err
}
