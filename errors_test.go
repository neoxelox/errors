package errors_test

import (
	goerr "errors"
	"fmt"
	"testing"

	"github.com/neoxelox/errors"
)

var ErrUserNotFound = errors.New("user %s not found")
var ErrCannotDeposit = errors.New("cannot deposit")

func Test(t *testing.T) {
	t.Parallel()

	err := view()
	if err == nil {
		t.FailNow()
	}

	if !ErrCannotDeposit.Is(err) {
		t.FailNow()
	}

	// nolint:forbidigo
	fmt.Printf("%+v", err)
}

func view() error {
	err := usecase()
	if err != nil {
		return ErrCannotDeposit.Raise().With("cannot add money to account %s", "ARN3107").Cause(err)
	}

	return nil
}

func usecase() error {
	err := repository()
	if err != nil {
		return err
	}

	return nil
}

func repository() error {
	err := goerr.New("other library error")
	if err != nil {
		return ErrUserNotFound.Raise("Alex").Extra(map[string]any{"userID": 310700, "accountID": "ARN3107"}).Cause(err)
	}

	return nil
}
