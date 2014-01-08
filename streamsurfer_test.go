package main

import (
	"fmt"
	"testing"
)

func TestSplitName(t *testing.T) {
	name, uri := splitName("the title http://uri")
	fmt.Printf("%s | %s\n", name, uri)
	name, uri = splitName("the title https://uri uri2part")
	fmt.Printf("%s | %s\n", name, uri)
	name, uri = splitName("http://uri the title")
	fmt.Printf("%s | %s\n", name, uri)
	name, uri = splitName("incorrect uri")
	fmt.Printf("%s | %s\n", name, uri)
}
