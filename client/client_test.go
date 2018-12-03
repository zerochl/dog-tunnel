package client

import (
	"testing"
	"log"
)

func TestClientAsServer(t *testing.T)  {
	StartClient()
}

func TestClientAsClient(t *testing.T)  {
	StartClient()
}

func TestTest(t *testing.T)  {
	var currIdMap map[string]int
	if currIdMap == nil {
		currIdMap = make(map[string]int)
	}
	currIdMap["test"]++
	log.Println("currIdMap:", currIdMap["test1"])
}