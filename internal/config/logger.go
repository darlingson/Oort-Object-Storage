package config

import (
	"log"
	"os"
)

var AppLogger *log.Logger

func InitLogger() {
	file, err := os.OpenFile(
		"myLOG.txt",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0666,
	)

	if err != nil {
		panic(err)
	}

	AppLogger = log.New(file, "OORT: ", log.LstdFlags|log.Lshortfile)
}