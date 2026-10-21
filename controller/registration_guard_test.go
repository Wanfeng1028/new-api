package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BotsRouter fork patch: registration email canonicalization must make
// plus aliases collapse onto the base mailbox for every provider, and
// Gmail dots collapse only for Gmail (dots are significant elsewhere).
func TestNormalizeEmailForRegistration(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plus alias hotmail", "user+1@hotmail.com", "user@hotmail.com"},
		{"plus alias outlook", "user+tag@outlook.com", "user@outlook.com"},
		{"plain address unchanged", "user@hotmail.com", "user@hotmail.com"},
		{"gmail dots via googlemail", "w.anfeng572@googlemail.com", "wanfeng572@gmail.com"},
		{"gmail dots plus alias and case", "W.Anfeng572+promo@Gmail.com", "wanfeng572@gmail.com"},
		{"dots significant outside gmail", "u.ser@163.com", "u.ser@163.com"},
		{"trim and lowercase", "  User@Example.COM ", "user@example.com"},
		{"not an email stays as-is", "not-an-email", "not-an-email"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeEmailForRegistration(tc.in)
			require.NotEmpty(t, got)
			assert.Equal(t, tc.want, got)
		})
	}

	// The anti-abuse invariant itself: an alias and its base mailbox
	// canonicalize to the same string.
	assert.Equal(t,
		NormalizeEmailForRegistration("user@hotmail.com"),
		NormalizeEmailForRegistration("user+9@hotmail.com"))
}
