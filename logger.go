package main

import (
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("spin360")

func init() {
	format := logging.MustStringFormatter(
		`spin360 %{color} %{shortfunc} %{level:.4s} %{shortfile}
%{id:03x}%{color:reset} %{message}`,
	)
	logging.SetFormatter(format)
}
