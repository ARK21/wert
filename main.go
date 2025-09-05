package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ARK21/wert/client"
	"github.com/ARK21/wert/domain"
)

type Exchanger interface {
	Exchange(ctx context.Context, ex domain.Exchange) (float64, error)
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			os.Exit(130) // 130 is common for SIGINT
		}
		log.Fatal(err)
	}
}

func run(args []string, out io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	exchanger, err := client.NewAPIClient(
		"https://sandbox-api.coinmarketcap.com",
		"b54bcf4d-1bca-4e8e-9a24-22ff2c3d462c",
	)
	if err != nil {
		return fmt.Errorf("cannot create fx client: %w", err)
	}

	return execute(ctx, args, exchanger, out)
}

func execute(ctx context.Context, args []string, exchanger Exchanger, out io.Writer) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: fxcli <amount> <from> <to>")
	}

	amount, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return fmt.Errorf("error parsing amount: %v", err)
	}
	if amount <= 0 {
		return fmt.Errorf("amount cannot be negative or zero: %.2f", amount)
	}
	from := strings.ToUpper(args[1])
	to := strings.ToUpper(args[2])

	fmt.Fprintln(out, "Exchange", amount, from, "to", to)

	exCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := exchanger.Exchange(exCtx, domain.Exchange{
		From:   from,
		Amount: amount,
		To:     to,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("cannot get exchange rate: %w", err)
	}

	fmt.Fprintln(out, "You received", res, to)

	return nil
}
