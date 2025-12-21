package main

import (
	"fmt"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"log"
)

func main() {
	comments, err := verify.GetComments("orch-go-oztz")
	if err != nil {
		log.Fatalf("GetComments failed: %v", err)
	}
	fmt.Printf("Successfully retrieved %d comments\n", len(comments))
	for _, c := range comments {
		fmt.Printf("  ID: %d, Author: %s, Content: %q\n", c.ID, c.Author, c.Text)
	}
}
