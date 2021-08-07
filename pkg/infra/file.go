package infra

import "io/ioutil"

func (x *Infrastructure) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}
