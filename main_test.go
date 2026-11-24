package main

import (
	"testing"
	"time"

	"github.com/Coop25/quotebot-go/config"
	"github.com/disgoorg/disgo/discord"
)

func TestBuildCommandsTransferSwitch(t *testing.T) {
	has := func(handlers map[string]commandHandler, creates []discord.ApplicationCommandCreate, name string) (bool, bool) {
		_, inMap := handlers[name]
		inList := false
		for _, c := range creates {
			if c.CommandName() == name {
				inList = true
			}
		}
		return inMap, inList
	}

	handlers, creates := buildCommands(config.Config{EnableTransferCommands: false})
	for _, name := range []string{"export", "import"} {
		if inMap, inList := has(handlers, creates, name); inMap || inList {
			t.Errorf("switch off: %q registered (handler=%v, command=%v)", name, inMap, inList)
		}
	}
	if inMap, inList := has(handlers, creates, "quote"); !inMap || !inList {
		t.Error("switch off: /quote should still be registered")
	}

	handlers, creates = buildCommands(config.Config{EnableTransferCommands: true})
	for _, name := range []string{"export", "import", "quote"} {
		if inMap, inList := has(handlers, creates, name); !inMap || !inList {
			t.Errorf("switch on: %q missing (handler=%v, command=%v)", name, inMap, inList)
		}
	}
}

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
