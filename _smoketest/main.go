package main

import (
    "fmt"
    "log"
    "github.com/dylan-conlin/orch-go/pkg/verify"
)

func main() {
    comments, err := verify.GetComments("orch-go-jz5")
    if err != nil {
        log.Fatalf("GetComments failed: %v", err)
    }
    fmt.Printf("Successfully retrieved %d comments\n", len(comments))
    for _, c := range comments {
        fmt.Printf("  ID: %d, Author: %s, Content: %q\n", c.ID, c.Author, c.Content)
    }
}
