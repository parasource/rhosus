package util

import (
	petname "github.com/dustinkirkland/golang-petname"
	"math/rand"
	"time"
)

func GenerateRandomName(words int) string {
	rand.Seed(time.Now().UnixNano())
	name := petname.Generate(words, "-")
	return name
}
