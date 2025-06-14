package main

import (
	"client/common"
	"client/view"
	"os"
)

func writeToFile(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data) // convert [20]byte to []byte
	return err
}

func main() {
	common.InitAppState()
	view.CreateMainWindow()
}
