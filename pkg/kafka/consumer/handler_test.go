package consumer

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSentinelErrors(t *testing.T) {
	t.Run("ErrSkipMessage is defined", func(t *testing.T) {
		assert.NotNil(t, ErrSkipMessage)
		assert.Equal(t, "skip message processing", ErrSkipMessage.Error())
	})

	t.Run("ErrPermanent is defined", func(t *testing.T) {
		assert.NotNil(t, ErrPermanent)
		assert.Equal(t, "permanent error", ErrPermanent.Error())
	})

	t.Run("errors can be wrapped and unwrapped", func(t *testing.T) {
		wrappedSkip := errors.Join(ErrSkipMessage, errors.New("additional context"))
		assert.True(t, errors.Is(wrappedSkip, ErrSkipMessage))

		wrappedPermanent := errors.Join(ErrPermanent, errors.New("additional context"))
		assert.True(t, errors.Is(wrappedPermanent, ErrPermanent))
	})
}
