package infra

import "fmt"

func (x *client) Stdout(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}
