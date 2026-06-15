package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	logFileName := "opexec.log"

	// open log file
	logFile, logFileErr := os.OpenFile(logFileName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if logFileErr != nil {
		log.Panic(logFileErr)
	}
	defer logFile.Close()

	if (len(os.Args) > 1) && os.Args[1] == "-debug" {
		// redirect all the output to file
		logWriter := io.MultiWriter(os.Stdout, logFile)

		// set log out put
		log.SetOutput(logWriter)

	} else {
		log.SetOutput(logFile)
	}

	log.Println("Log setup, starting to read standard input")
	Stdin := bufio.NewReader(os.Stdin)
	StandardInput, _ := Stdin.ReadString('\n')
	StandardInput = strings.TrimSuffix(StandardInput, "\n")

	log.Printf("Standard input was: %s\n", StandardInput)

	var RequestPayload Request

	//var target map[string]any

	err := json.Unmarshal([]byte(StandardInput), &RequestPayload)
	if err != nil {
		log.Printf("Received from stdin was: %s\n", StandardInput)
		log.Printf("Unable to marshal stdin as JSON due to %s\n", err)
		os.Exit(1)
	}

	Reference := RequestPayload.IDs[0]
	log.Printf("Found out the Reference is : %s\n", Reference)

	OPToken := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	if OPToken == "" {
		log.Println("No environment variable value found for OP_SERVICE_ACCOUNT_TOKEN")
		os.Exit(2)
	}

	log.Println("Got the envrironment variable.")

	client, err := onepassword.NewClient(
		context.TODO(),
		onepassword.WithServiceAccountToken(OPToken),
		onepassword.WithIntegrationInfo("opexec for OpenClaw", "v1.0.0"),
	)
	if err != nil {
		log.Printf("Unable to connect to 1Password due to %s\n", err)
		os.Exit(3)
	}

	log.Println("Connected to 1Password.")

	Value, err := client.Secrets().Resolve(context.TODO(), Reference)
	if err != nil {
		log.Printf("Reference: %s\n", Value)
		log.Printf("Unable to retrieve secret due to %s\n", err)
		os.Exit(4)
	}

	log.Println("Got the secret value.")

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
	log.Println("Outputting the response.")

	fmt.Println(string(ResponseData))
}
