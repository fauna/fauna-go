package fauna

import (
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var randomSeeded = false
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func arrayContains[T comparable](a []T, o T) bool {
	for _, t := range a {
		if o == t {
			return true
		}
	}
	return false
}

func getRandomString(length int) string {
	if !randomSeeded {
		rand.Seed(time.Now().UnixNano())
		randomSeeded = true
	}

	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func isLaunchedByDebugger() bool {
	gopsOut, err := exec.Command("gops", strconv.Itoa(os.Getppid())).Output()
	if err == nil && strings.Contains(string(gopsOut), "\\dlv.exe") {
		return true
	}
	return false
}
