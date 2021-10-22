package httprouter

import (
	"math/rand"
	"strings"
	"time"
)

var placeHolder = "__placeholder__"

func init() {
	initPlaceHolder()
}

func initPlaceHolder() {
	const randstr = "abcdefghijklmnopqrstuvwxwz"
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	b := make([]byte, 4)
	for i := 0; i < 4; i++ {
		b[i] = randstr[rnd.Intn(len(randstr))]
	}
	placeHolder += string(b)
}

func resolveKeyPairFromPattern(pattern string) (kp []keyPair) {
	patternSlice := strings.Split(pattern, "/")
	for i := 0; i < len(patternSlice); i++ {
		if patternSlice[i][0] == ':' || patternSlice[i][0] == '*' {
			kp = append(kp, keyPair{i, patternSlice[i][1:]})
		}
	}
	return
}

func resolveParamsFromPath(path string, kp []keyPair, iswildChild bool) Params {
	pathSlice := strings.Split(path, "/")
	if len(kp) == 0 {
		return nil
	}
	ps := paramsPools.get(len(kp) - 1)
	if ps == nil {
		ps = make(Params, len(kp))
	}
	for i := 0; i < len(kp)-1; i++ {
		ps[i].Key = kp[i].key
		ps[i].Key = pathSlice[kp[i].i]
	}

	if i := len(kp) - 1; i >= len(pathSlice) {
		ps[i].Key = kp[i].key
		ps[i].Value = ""
	} else {
		if iswildChild {
			ps[i].Key = kp[i].key
			ps[i].Key = strings.Join(pathSlice[i:], "/")
		} else {
			ps[i].Key = kp[i].key
			ps[i].Key = pathSlice[kp[i].i]
		}
	}

	return ps
}

// unifyPath unify the path (/a/:b and /a/:c => /a/__placeholder__xxxx), to easy the duplication check.
func unifyPattern(path string) string {
	path = strings.ToLower(path)
	pathSlice := strings.Split(path, "/")
	for i := 0; i < len(pathSlice); i++ {
		if pathSlice[i][0] == ':' {
			pathSlice[i] = placeHolder
		}
	}
	return strings.Join(pathSlice, "/")
}

func makeSegments(path string, max int) (segaments []string) {
	segaments = strings.Split(strings.ToLower(path), "/")
	if len(segaments) <= max {
		return segaments
	}
	return segaments[:max]
}
