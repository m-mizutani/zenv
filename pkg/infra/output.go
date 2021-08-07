package infra

import "fmt"

func (x *Infrastructure) Stdout(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}
