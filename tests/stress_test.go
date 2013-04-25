package main

import (
	"../statsd"
	"encoding/json"
	"io"
	"os"
)

type RequestLog struct {
	Url         string  `json:url`
	ProcessTime float64 `json:proces_time`
	CreateTime  int64   `json:create_time`
}

func main() {
	client = statsd.NewStatsdClient("localhost", 8125)
	inputFile, inputError = os.Open("data/requestlog_vip18_vipacceleratecore.log.20130420")
	if inputError != nil {
		fmt.Printf("an error accurred on opening the input file.")
		return
	}
	defer inputFile.Close()
	inputReader := bufio.NewReader(inputFile)
	for i := 0; i < 10; i++ {
		inputString, readerError := inputReader.ReadString('\n')
		if readerError == io.EOF {
			return
		}
		var msg RequestLog
		err := json.Unmarshal(inputString, &msg)
		if err != nil {
			fmt.Println("error: ", err)
		}
		fmt.Println("msg: ", msg)
	}
}
