package main

import (
	"bytes"
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"io"
)

func collectionGroupQuery(w io.Writer, projectID string) error {
	ctx := context.Background()

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("firestore.NewClient: %v", err)
	}
	defer client.Close()

	it := client.CollectionGroup("Dogs").Where("name", "==", "dalmation").Documents(ctx)
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("documents iterator: %v", err)
		}
		fmt.Println("Value is : ", doc.Data(), doc.Data()["name"])
		fmt.Println("Testing")
	}

	return nil

}

func main() {
	var projectID = "sandbox-20220906-9lrhsl"
	fmt.Println("Hello, World!")

	buf := &bytes.Buffer{}
	if err := collectionGroupQuery(buf, projectID); err != nil {
		fmt.Println(" Error in code!!!  ")
	}

}
