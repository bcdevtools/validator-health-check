package utils

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsValoperAddressFormat(t *testing.T) {
	//goland:noinspection SpellCheckingInspection
	require.True(t, IsValoperAddressFormat("cosmosvaloper18ruzecmqj9pv8ac0gvkgryuc7u004te9rh7w5s"))
}
