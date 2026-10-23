package main

import (
	"syscall"

	"github.com/wtsi-hgi/softpack/cmd"
)

func init() {
	syscall.Umask(0)
}

func main() {
	cmd.Execute()
}
