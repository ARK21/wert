package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/ARK21/wert/domain"
)

type fakeExchange struct {
	got domain.Exchange
	res float64
	err error
}

func (f *fakeExchange) Exchange(ctx context.Context, e domain.Exchange) (float64, error) {
	f.got = e
	return f.res, f.err
}

func TestExecute_OK(t *testing.T) {
	f := &fakeExchange{res: 42}
	var buf bytes.Buffer

	err := execute(context.Background(), []string{"123.45", "usd", "btc"}, f, &buf)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	out := buf.String()
	if f.got.Amount != 123.45 || f.got.From != "USD" || f.got.To != "BTC" {
		t.Fatalf("got wrong exchange input: %+v", f.got)
	}
	if !contains(out, "You received 42 BTC") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestExecute_ErrorFromUsecase(t *testing.T) {
	f := &fakeExchange{err: errors.New("boom")}
	var buf bytes.Buffer

	err := execute(context.Background(), []string{"1", "usd", "btc"}, f, &buf)
	if err == nil || !contains(err.Error(), "boom") {
		t.Fatalf("want error containing 'boom', got %v", err)
	}
}

func contains(s, sub string) bool { return bytes.Contains([]byte(s), []byte(sub)) }
