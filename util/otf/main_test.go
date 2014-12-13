package otf

import (
	"fmt"
	"testing"
)

func TestCreateKeyPair(t *testing.T) {
	var err error
	var crt, key string
	if crt, key, err = CreateKeyPair("example.org"); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("crt: %s, key: %s\n", crt, key)
}
