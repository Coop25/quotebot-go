package commands

import (
	"testing"

	"github.com/disgoorg/snowflake/v2"
)

func TestIsOwner(t *testing.T) {
	owner := snowflake.ID(123456789012345678)

	if !isOwner(owner, "123456789012345678") {
		t.Error("the owner should be allowed")
	}
	if isOwner(snowflake.ID(999), "123456789012345678") {
		t.Error("a different user should be refused")
	}
	if isOwner(owner, "") {
		t.Error("everyone should be refused when OWNER_ID is not set")
	}
}
