package ocihandlers

import (
	"fmt"
	"net/http"
)

func Tokens(w http.ResponseWriter, r *http.Request) {
	fmt.Println("tokens")
}
