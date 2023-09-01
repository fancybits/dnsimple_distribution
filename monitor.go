package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dnsimple/dnsimple-go/dnsimple"
)

type DistributionCheck struct {
	StartAt  time.Time     `json:"start_at"`
	Duration time.Duration `json:"duration"`
	Checks   int           `json:"checks"`
	Hostname string        `json:"hostname"`
	Deleted  bool          `json:"deleted"`
	Err      error         `json:"error"`
}

type ErrTimeout struct {
	Duration time.Duration
}

func (e *ErrTimeout) Error() string {
	return fmt.Sprintf("Timeout after %s", e.Duration)
}

func (e *ErrTimeout) Timeout() bool   { return true }
func (e *ErrTimeout) Temporary() bool { return true }

func Monitor(ctx context.Context, client *dnsimple.Client, accountID string, domain string, interval, poll, timeout time.Duration) chan *DistributionCheck {
	ch := make(chan *DistributionCheck)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				go func() {
					ctx, timeoutCancel := context.WithTimeoutCause(ctx, timeout, &ErrTimeout{timeout})
					defer timeoutCancel()

					c, err := Check(ctx, client, accountID, domain, poll)
					c.Err = err
					ch <- c
				}()
			}
		}
	}()

	return ch
}

func Check(ctx context.Context, client *dnsimple.Client, accountID string, domain string, poll time.Duration) (*DistributionCheck, error) {
	var c DistributionCheck

	start := time.Now()
	c.StartAt = start
	c.Hostname = fmt.Sprintf("_distribution_check_%s", start.Format("20060102150405"))
	attrs := dnsimple.ZoneRecordAttributes{
		Type:    "TXT",
		Name:    &c.Hostname,
		Content: fmt.Sprintf("distribution-check: %s", start.Format(time.RFC3339)),
	}

	record, err := client.Zones.CreateRecord(ctx, accountID, domain, attrs)
	if err != nil {
		return &c, err
	}

	recordID := record.Data.ID

	// Record the duration of the check
	defer func() {
		c.Duration = time.Since(start)
	}()

	// Delete the record when we're done
	defer func() {
		_, err := client.Zones.DeleteRecord(context.Background(), accountID, domain, recordID)
		c.Deleted = err == nil
	}()

	for {
		select {
		case <-ctx.Done():
			return &c, context.Cause(ctx)
		case <-time.After(poll):
			c.Checks++
			response, err := client.Zones.CheckZoneRecordDistribution(ctx, accountID, domain, recordID)
			if err != nil {
				return &c, err
			}
			if response.Data.Distributed {
				return &c, nil
			}
		}
	}
}
