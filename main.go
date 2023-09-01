package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/dnsimple/dnsimple-go/dnsimple"
	kitlog "github.com/go-kit/log"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

func main() {
	logger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))

	fs := ff.NewFlags("dnsimple_distribution")

	var (
		poll     = fs.DurationLong("poll", 2*time.Second, "interval between checks")
		timeout  = fs.DurationLong("timeout", 10*time.Minute, "timeout for check")
		interval = fs.DurationLong("interval", time.Minute, "interval between checks")
		domain   = fs.StringLong("domain", "", "domain to check")
		token    = fs.StringLong("token", "", "dnsimple API Access Token")
		_        = fs.StringLong("config", "", "config file (optional)")
	)

	err := ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("DD"),
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFile(".config"),
		ff.WithConfigAllowMissingFile(),
		ff.WithConfigFileParser(ff.PlainParser),
	)

	if errors.Is(err, ff.ErrHelp) {
		fmt.Fprint(os.Stderr, ffhelp.Flags(fs))
		os.Exit(0)
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *token == "" {
		fmt.Fprintf(os.Stderr, "error: token is required\n")
		fmt.Fprint(os.Stderr, ffhelp.Flags(fs))
		os.Exit(1)
	}

	if *domain == "" {
		fmt.Fprintf(os.Stderr, "error: domain is required\n")
		fmt.Fprint(os.Stderr, ffhelp.Flags(fs))
		os.Exit(1)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	// Listen for signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Wait for a signal
	go func() {
		sig := <-c
		cancel(fmt.Errorf("received signal: %s", sig))
	}()

	tc := dnsimple.StaticTokenHTTPClient(ctx, *token)
	tc.Timeout = time.Minute
	client := dnsimple.NewClient(tc)

	// get the current authenticated account
	whoamiResponse, err := client.Identity.Whoami(ctx)
	if err != nil {
		fmt.Printf("Whoami() returned error: %v\n", err)
		os.Exit(1)
	}

	// either assign the account ID or fetch it from the response
	// if you are authenticated with an account token
	accountID := strconv.FormatInt(whoamiResponse.Data.Account.ID, 10)

	// Sleep until the next minute starts
	delay := time.Until(time.Now().Truncate(*interval).Add(*interval))
	_ = logger.Log("msg", "Sleeping until next interval", "delay", delay)

	select {
	case <-ctx.Done():
		return
	case <-time.After(delay):
	}

	ch := Monitor(ctx, client, accountID, *domain, *interval, *poll, *timeout)

	for {
		select {
		case <-ctx.Done():
			// Drain the queue if it's there
			select {
			case <-ch:
			default:
			}
			return
		case c := <-ch:
			l := kitlog.With(logger, "at", c.StartAt.Truncate(time.Second),
				"duration", c.Duration.Truncate(time.Millisecond),
				"checks", c.Checks, "deleted", c.Deleted)
			if c.Err != nil {
				l = kitlog.With(l, "error", c.Err)
			}
			_ = l.Log()

		}
	}
}
