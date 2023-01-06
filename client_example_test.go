package fauna_test

import (
	"fmt"
	"log"

	"github.com/fauna/fauna-go"
)

func ExampleClient() {
	client := fauna.NewClient("secret", fauna.URL(fauna.EndpointLocal))

	var result float32
	_, queryErr := client.Query(`Math.abs(12e5)`, nil, &result)
	if queryErr != nil {
		log.Fatalf(queryErr.Error())
	}

	fmt.Printf("%0.f", result)
	// Output: 1200000
}
