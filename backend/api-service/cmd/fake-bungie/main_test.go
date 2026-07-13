package main

import "testing"

func TestLoopbackAddress(t *testing.T) {
	for _, address := range []string{"127.0.0.1:8090", "[::1]:8090", "localhost:8090"} {
		if !loopbackAddress(address) {
			t.Errorf("loopbackAddress(%q) = false", address)
		}
	}
	for _, address := range []string{"0.0.0.0:8090", "192.0.2.1:8090", ":8090", "bad"} {
		if loopbackAddress(address) {
			t.Errorf("loopbackAddress(%q) = true", address)
		}
	}
}
