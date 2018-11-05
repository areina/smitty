package agent

import (
	"fmt"
	"io/ioutil"
	"launchpad.net/goyaml"
	"log"
	"os"
)

var logger *log.Logger

func SetFileLogger() {
	logFile, err := os.OpenFile(Settings.LogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		panic("Error opening log file")
	}
	logger = log.New(logFile, "", log.Ldate|log.Lmicroseconds|log.Lshortfile)
}

func Log(value ...interface{}) {
	logger.Println(value...)
}

func Fatal(value ...interface{}) {
	log.Println(value...)
	logger.Fatal(value...)
}

func Debug(value ...interface{}) {
	if Settings.Verbose == true {
		Log(value...)
	}
}

func WriteYaml(path string, obj interface{}) error {
	data, err := goyaml.Marshal(obj)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_SYNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)

	if err != nil {
		return err
	}

	error := os.Rename(tmp, path)
	fmt.Println(error)
	return error
}

func ReadYaml(path string, obj interface{}) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return goyaml.Unmarshal(data, obj)
}

func ComposeRedisAddress(ip string, port string) (address string) {
	address = fmt.Sprint(ip, ":", port)
	return address
}
