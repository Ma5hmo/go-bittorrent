package common

import (
	"fmt"
	"math/rand"
)

var AppState struct {
	PeerID [20]byte
	Port   uint16
}

func Init() {
	copy(AppState.PeerID[:], []byte(fmt.Sprintf("-GT001-%012d", rand.Int63())))
}
