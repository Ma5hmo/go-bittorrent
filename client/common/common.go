package common

import (
	"fmt"
	"math/rand"
)

var AppState struct {
	PeerID                [20]byte
	Port                  uint16
	IsTrafficAESEncrypted bool
}

func InitAppState() {
	AppState.Port = 6881
	copy(AppState.PeerID[:], []byte(fmt.Sprintf("-GT001-%012d", rand.Int63())))
	AppState.IsTrafficAESEncrypted = true
}
