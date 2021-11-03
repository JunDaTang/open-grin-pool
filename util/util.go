package util
import (
	"fmt"
	"reflect"
	"runtime"
)

// Gogogo 带panic处理的启动协程
func Gogogo(f interface{}, vals ...interface{}) {
	t := reflect.TypeOf(f)
	if t.Kind() != reflect.Func {
		return
	}
	fn := reflect.ValueOf(f)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				bf := make([]byte, 2048)
				n := runtime.Stack(bf, false)
				fmt.Println(string(bf[:n]))
			}
		}()
		args := []reflect.Value{}
		for _, v := range vals {
			args = append(args, reflect.ValueOf(v))
		}
		fn.Call(args)
	}()
}

