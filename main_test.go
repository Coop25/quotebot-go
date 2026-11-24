package main

import (
	"testing"
	"time"
)

func TestCheckCooldown(t *testing.T) {
	key := "user1quote"
	t.Cleanup(func() { delete(commandCooldowns, key) })

	if _, onCooldown := checkCooldown(key, time.Minute); onCooldown {
		t.Fatal("first use should not be on cooldown")
	}
	remaining, onCooldown := checkCooldown(key, time.Minute)
	if !onCooldown {
		t.Fatal("second use within the cooldown should be blocked")
	}
	if remaining <= 0 || remaining > time.Minute {
		t.Fatalf("unexpected remaining duration %v", remaining)
	}

	// A refunded cooldown (failed command) allows an immediate retry.
	clearCooldown(key)
	if _, onCooldown := checkCooldown(key, time.Minute); onCooldown {
		t.Fatal("use after clearCooldown should be allowed")
	}

	// Once the cooldown has elapsed the command is allowed again.
	commandCooldowns[key] = time.Now().Add(-2 * time.Minute)
	if _, onCooldown := checkCooldown(key, time.Minute); onCooldown {
		t.Fatal("use after the cooldown expired should be allowed")
	}
}
