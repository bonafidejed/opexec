package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/1Password/connect-sdk-go/connect"
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

func GetSecretFromOP(ref string) (string, error) {
	client, err := onepassword.NewClient(
		context.TODO(),
		onepassword.WithServiceAccountToken(os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")),
		onepassword.WithIntegrationInfo("opexec for OpenClaw", "v1.0.0"),
	)
	if err != nil {
		return "", fmt.Errorf("1Password unable to connect due to: %w", err)
	}

	log.Println("1Password connected.")

	val, err := client.Secrets().Resolve(context.TODO(), ref)
	if err != nil {
		return "", fmt.Errorf("1Password unable to retrieve the secret due to: %w", err)
	}

	return val, nil
}

func GetSecretFromConnect(ref string) (string, error) {
	var refVault string
	var refItem string
	var refSection any
	var refField string

	if strings.HasPrefix(ref, "op://") {
		refSplit := strings.Split(strings.TrimLeft(ref, "op://"), "/")
		switch len(refSplit) {
		case 3:
			refVault = refSplit[0]
			refItem = refSplit[1]
			refSection = nil
			refField = refSplit[2]
		case 4:
			refVault = refSplit[0]
			refItem = refSplit[1]
			refSection = refSplit[2]
			refField = refSplit[3]
		default:
			return "", errors.New("OP Connect cannot continue, that is not a valid 1Password referecne string")
		}
	} else {
		return "", errors.New("OP Connect cannot continue, that is not a valid 1Password referecne string")
	}

	conn, err := connect.NewClientFromEnvironment()
	if err != nil {
		return "", fmt.Errorf("OP Connect could not connect to the server because of: %w", err)
	}
	item, err := conn.GetItem(refItem, refVault)
	if err != nil {
		return "", fmt.Errorf("OP Connect could not retrieve the item because of: %w", err)
	}

	log.Println("OP Connect connected.")

	var idxs []int
	for i, field := range item.Fields {
		if (field.Label == refField) && ((field.Section == refSection) || refSection == nil) {
			idxs = append(idxs, i)
		} else {
			//fmt.Printf("It's not %s\n", field.ID)
		}
	}

	switch {
	case len(idxs) == 0:
		return "", errors.New("OP Connect that [section/]field does not exist on that item.")
	case len(idxs) > 1:
		return "", errors.New("OP Connect more than one field matched the secret reference.")
	}

	idx := idxs[0]

	return item.Fields[idx].Value, nil
}

func main() {
	log.SetOutput(os.Stderr)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-file":
			lfName := "opexec.log"
			lf, lfErr := os.OpenFile(lfName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
			if lfErr != nil {
				log.Panic(lfErr)
			}
			defer lf.Close()
			log.SetOutput(lf)
		case "-screen":
		default:
			log.SetOutput(io.Discard)
		}
	} else {
		log.SetOutput(io.Discard)
	}

	log.Println("Log setup, starting to read standard input")

	in := bufio.NewReader(os.Stdin)
	inStr, _ := in.ReadString('\n')
	inStr = strings.TrimSuffix(inStr, "\n")

	log.Printf("Standard input was: %s\n", inStr)

	var req Request
	err := json.Unmarshal([]byte(inStr), &req)
	if err != nil {
		log.Printf("Unable to marshal stdin as JSON due to %s\n", err)
		os.Exit(99)
	}

	if len(req.IDs) < 1 {
		log.Printf("Invalid input JSON, did not find any ids to process.")
		os.Exit(99)
	}
	ref := req.IDs[0]

	log.Printf("Found out the Reference is : %s\n", ref)

	var val string
	if (os.Getenv("OP_CONNECT_HOST") != "") && (os.Getenv("OP_CONNECT_TOKEN") != "") {
		val, err = GetSecretFromConnect(ref)
		if (err != nil) && (os.Getenv("OP_SERVICE_ACCOUNT_TOKEN") != "") {
			fmt.Printf("Error retrieving secret: %s", err)
			val, err = GetSecretFromOP(ref)
		}
	} else if os.Getenv("OP_SERVICE_ACCOUNT_TOKEN") != "" {
		val, err = GetSecretFromOP(ref)
	} else {
		log.Println("Environment Variable(s) not found! To use Connect, set OP_CONNECT_HOST and OP_CONNECT_TOKEN. To use 1Password, set OP_SERVICE_ACCOUNT_TOKEN.")
		os.Exit(99)
	}
	if err != nil {
		log.Printf("Error retrieving secret: %s", err)
		os.Exit(89)
	}

	log.Printf("Got the secret value.")

	resp := Response{
		ProtocolVersion: 1,
		Values: map[string]string{
			ref: val,
		},
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to generate JSON for output: %v", err)
	}
	log.Println("Outputting the response.")

	fmt.Println(string(respData))
}
