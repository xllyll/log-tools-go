package xip

import (
	"fmt"
	"testing"
)

func TestGetLocalIP(t *testing.T) {
	ip := GetLocalIP()
	fmt.Println(ip)
}
