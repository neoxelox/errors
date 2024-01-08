package errors_test

import (
	goerrors "errors"
	"fmt"
	"testing"

	"github.com/neoxelox/errors"
)

var ErrOtherLibrary = goerrors.New("other library error")
var ErrUserNotFound = errors.New("user %s not found")
var ErrCannotDeposit = errors.New("cannot deposit")

func TestPrint(t *testing.T) {
	t.Parallel()

	err := view()
	if err == nil {
		t.FailNow()
	}

	if !ErrCannotDeposit.Is(err) {
		t.FailNow()
	}

	cerr, ok := err.(*errors.Error)
	if !ok {
		t.FailNow()
	}

	if !cerr.Has(ErrOtherLibrary) {
		t.FailNow()
	}

	if !cerr.Has(ErrUserNotFound) {
		t.FailNow()
	}

	if !cerr.Has(ErrCannotDeposit) {
		t.FailNow()
	}

	// nolint:forbidigo
	fmt.Printf("%+v", err)
}

func TestString(t *testing.T) {
	t.Parallel()

	err := view()
	if err == nil {
		t.FailNow()
	}

	if !ErrCannotDeposit.Is(err) {
		t.FailNow()
	}

	cerr, ok := err.(*errors.Error)
	if !ok {
		t.FailNow()
	}

	if !cerr.Has(ErrOtherLibrary) {
		t.FailNow()
	}

	if !cerr.Has(ErrUserNotFound) {
		t.FailNow()
	}

	if !cerr.Has(ErrCannotDeposit) {
		t.FailNow()
	}

	// nolint:forbidigo
	fmt.Printf("%s", cerr.StringReport())
}

func TestSentry(t *testing.T) {
	t.Parallel()

	err := view()
	if err == nil {
		t.FailNow()
	}

	if !ErrCannotDeposit.Is(err) {
		t.FailNow()
	}

	cerr, ok := err.(*errors.Error)
	if !ok {
		t.FailNow()
	}

	if !cerr.Has(ErrOtherLibrary) {
		t.FailNow()
	}

	if !cerr.Has(ErrUserNotFound) {
		t.FailNow()
	}

	if !cerr.Has(ErrCannotDeposit) {
		t.FailNow()
	}

	// nolint:forbidigo
	fmt.Printf("%+v", cerr.SentryReport())
}

func view() error {
	err := usecase()
	if err != nil {
		return ErrCannotDeposit.Raise().
			With("cannot add money to account %s", "ARN3107").Tags(map[string]any{"apiVersion": 2}).Cause(err)
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
	err := ErrOtherLibrary
	if err != nil {
		return ErrUserNotFound.Raise("Alex").
			Extra(map[string]any{"userID": 310700, "accountID": "ARN3107"}).Cause(err)
	}

	return nil
}
