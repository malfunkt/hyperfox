package gencert

import (
	"fmt"
	"testing"
)

func TestCreateKeyPair(t *testing.T) {
	crt, key, err := CreateKeyPair("example.ORG")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("crt: %s, key: %s\n", crt, key)

	crt, key, err = CreateKeyPair("büchEr.example.com")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("crt: %s, key: %s\n", crt, key)
}
