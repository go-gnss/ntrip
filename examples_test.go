package ntrip

import (
	"context"
	"fmt"
)

func ExampleGetSourcetable() {
	ctx := context.Background()
	url := "http://auscors.ga.gov.au:2101"

	mapping, warnings, err := GetSourcetable(ctx, url)

	fmt.Println(mapping, warnings, err)
}
