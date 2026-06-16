package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/1Password/connect-sdk-go/connect"
	connectop "github.com/1Password/connect-sdk-go/onepassword" // <-- Add this alias
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

func GetSecretFromOP(ctx context.Context, refs []string) (map[string]string, error) {
	vals := make(map[string]string)
	client, err := onepassword.NewClient(
		ctx,
		onepassword.WithServiceAccountToken(os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")),
		onepassword.WithIntegrationInfo("opexec for OpenClaw", "v1.0.0"),
	)
	if err != nil {
		return vals, fmt.Errorf("1Password unable to connect due to: %w", err)
	}

	log.Println("1Password connected.")

	for _, ref := range refs {
		val, err := client.Secrets().Resolve(context.TODO(), ref)
		if err != nil {
			return vals, fmt.Errorf("1Password unable to retrieve the secret for %s due to: %w", ref, err)
		}
		vals[ref] = val
	}

	return vals, nil
}

func GetSecretFromConnect(refs []string) (map[string]string, error) {
	vals := make(map[string]string)

	conn, err := connect.NewClientFromEnvironment()
	if err != nil {
		return vals, fmt.Errorf("OP Connect could not connect to the server because of: %w", err)
	}

	log.Println("OP Connect connected.")

	itemCache := make(map[string]*connectop.Item)

	for _, ref := range refs {
		var refVault string
		var refItem string
		var refSection string
		var refField string

		if strings.HasPrefix(ref, "op://") {
			refSplit := strings.Split(strings.TrimLeft(ref, "op://"), "/")
			switch len(refSplit) {
			case 3:
				refVault = refSplit[0]
				refItem = refSplit[1]
				refSection = ""
				refField = refSplit[2]
			case 4:
				refVault = refSplit[0]
				refItem = refSplit[1]
				refSection = refSplit[2]
				refField = refSplit[3]
			default:
				return vals, fmt.Errorf("OP Connect cannot continue, %s is not a valid 1Password reference string.", ref)
			}
		} else {
			return vals, fmt.Errorf("OP Connect cannot continue, %s is not a valid 1Password reference string.", ref)
		}

		cacheKey := refVault + "/" + refItem

		item, exists := itemCache[cacheKey]
		if !exists {
			var apiErr error
			// If it doesn't exist, fetch it from the Connect server
			item, apiErr = conn.GetItem(refItem, refVault)
			if apiErr != nil {
				return vals, fmt.Errorf("OP Connect could not retrieve the item for %s because of: %w", ref, apiErr)
			}
			// Store the retrieved item in the cache for subsequent loops
			itemCache[cacheKey] = item
		}

		var idxs []int
		for i, field := range item.Fields {
			if field.Label != refField && field.ID != refField {
				continue
			}
			if refSection == "" {
				if field.Section != nil {
					if field.Section.ID != "add more" {
						continue
					}
				}
			} else {
				if field.Section == nil {
					continue
				}
				if field.Section.Label != refSection && field.Section.ID != refSection {
					continue
				}
			}
			idxs = append(idxs, i)
		}

		switch {
		case len(idxs) == 0:
			return vals, fmt.Errorf("OP Connect the [section/]field does not exist on %s.", ref)
		case len(idxs) > 1:
			return vals, fmt.Errorf("OP Connect more than one field matched %s.", ref)
		}

		idx := idxs[0]

		vals[ref] = item.Fields[idx].Value
	}

	return vals, nil

}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error

	log.SetOutput(os.Stderr)
	logToFile := flag.Bool("file", false, "write logs to opexec.log in addition to stderr")
	flag.Parse()
	if *logToFile {
		f, err := os.OpenFile("opexec.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening log file: %v", err)
		}
		defer f.Close()
		multiWriter := io.MultiWriter(os.Stderr, f)
		log.SetOutput(multiWriter)
	}

	log.Println("Log setup, starting to read standard input")

	var req Request
	err = json.NewDecoder(os.Stdin).Decode(&req)
	if err != nil {
		log.Fatalf("Unable to decode stdin as JSON: %v", err)
	}
	refs := req.IDs

	log.Printf("Found out the list of References is : %s\n", refs)

	var vals map[string]string
	if (os.Getenv("OP_CONNECT_HOST") != "") && (os.Getenv("OP_CONNECT_TOKEN") != "") {
		vals, err = GetSecretFromConnect(refs)
		if (err != nil) && (os.Getenv("OP_SERVICE_ACCOUNT_TOKEN") != "") {
			log.Printf("Error retrieving secret: %s", err)
			vals, err = GetSecretFromOP(ctx, refs)
		}
	} else if os.Getenv("OP_SERVICE_ACCOUNT_TOKEN") != "" {
		vals, err = GetSecretFromOP(ctx, refs)
	} else {
		log.Fatalln("Environment Variable(s) not found! To use Connect, set OP_CONNECT_HOST and OP_CONNECT_TOKEN. To use 1Password, set OP_SERVICE_ACCOUNT_TOKEN.")
	}
	if err != nil {
		log.Fatalf("Error retrieving secret: %s", err)
	}

	log.Printf("Got the secret values.")

	resp := Response{
		ProtocolVersion: 1,
		Values:          vals,
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Failed to generate JSON for output: %v", err)
	}
	log.Println("Outputting the response.")

	fmt.Println(string(respData))
}
