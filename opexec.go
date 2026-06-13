package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/1password/onepassword-sdk-go"
)

type Request struct {
	ProtocolVersion int      `json:"protocolVersion"`
	Provider        string   `json:"provider"`
	IDs             []string `json:"ids"`
}

type Response struct {
	ProtocolVersion int               `json:"protocolVersion"`
	Values          map[string]string `json:"values"`
}

// type Response struct {
// 	ProtocolVersion int		`json:"protocolVersion"`
// 	Values struct {

// 	} 						`json:"values"`
// }

func main() {
	Stdin := bufio.NewReader(os.Stdin)
	StandardInput, _ := Stdin.ReadString('\n')
	StandardInput = strings.TrimSuffix(StandardInput, "\n")

	var RequestPayload Request

	//var target map[string]any

	err := json.Unmarshal([]byte(StandardInput), &RequestPayload)
	if err != nil {
		log.Printf("Received from stdin was: %s\n", StandardInput)
		log.Printf("Unable to marshal stdin as JSON due to %s\n", err)
		os.Exit(1)
	}

	Reference := RequestPayload.IDs[0]

	OPToken := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	if OPToken == "" {
		log.Println("No environment variable value found for OP_SERVICE_ACCOUNT_TOKEN")
		os.Exit(2)
	}

	client, err := onepassword.NewClient(
		context.TODO(),
		onepassword.WithServiceAccountToken(OPToken),
		onepassword.WithIntegrationInfo("opexec for OpenClaw", "v1.0.0"),
	)
	if err != nil {
		log.Printf("Unable to connect to 1Password due to %s\n", err)
		os.Exit(3)
	}
	Value, err := client.Secrets().Resolve(context.TODO(), Reference)
	if err != nil {
		log.Printf("Reference: %s\n", Value)
		log.Printf("Unable to retrieve secret due to %s\n", err)
		os.Exit(4)
	}

	ResponsePayload := Response{
		ProtocolVersion: 1,
		Values: map[string]string{
			Reference: Value,
		},
	}

	ResponseData, err := json.Marshal(ResponsePayload)
	if err != nil {
		log.Printf("Failed to generate JSON for output: %v", err)
	}

	fmt.Println(string(ResponseData))
}
