package bitwarden

import (
	"errors"
	"fmt"
)

type Status uint8

const (
	StatusUnauthenticated = Status(0)
	StatusLocked          = Status(1)
	StatusUnlocked        = Status(2)
)

var (
	ErrIllegalStatus = errors.New("illegal status")
)

func (this Status) IsUsable() bool {
	switch this {
	case StatusUnlocked:
		return true
	default:
		return false
	}
}

func (this Status) MarshalText() ([]byte, error) {
	switch this {
	case StatusUnauthenticated:
		return []byte("unauthenticated"), nil
	case StatusLocked:
		return []byte("locked"), nil
	case StatusUnlocked:
		return []byte("unlocked"), nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrIllegalStatus, this)
	}
}

func (this *Status) UnmarshalText(in []byte) error {
	switch string(in) {
	case "unauthenticated":
		*this = StatusUnauthenticated
		return nil
	case "locked":
		*this = StatusLocked
		return nil
	case "unlocked":
		*this = StatusUnlocked
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrIllegalStatus, string(in))
	}
}

func (this Status) String() string {
	switch this {
	case StatusUnauthenticated:
		return "unauthenticated"
	case StatusLocked:
		return "locked"
	case StatusUnlocked:
		return "unlocked"
	default:
		return fmt.Sprintf("illegal-status-%d", this)
	}
}
