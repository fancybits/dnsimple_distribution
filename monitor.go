package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dnsimple/dnsimple-go/dnsimple"
)

type DistributionCheck struct {
	Timing   *Timing `json:"timing"`
	Checks   int     `json:"checks"`
	Hostname string  `json:"hostname"`
	Created  bool    `json:"created"`
	Deleted  bool    `json:"deleted"`
	Err      error   `json:"error"`
	Timings  struct {
		Create *Timing `json:"create"`
		Check  Timings `json:"check"`
		Delete *Timing `json:"delete"`
	} `json:"timings"`
}

type ErrTimeout struct {
	Duration time.Duration
}

func (e *ErrTimeout) Error() string   { return fmt.Sprintf("Timeout after %s", e.Duration) }
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

	c.Timing = NewTiming()
	defer func() { c.Timing.Stop() }()

	c.Hostname = fmt.Sprintf("_distribution_check_%s", c.Timing.StartAt.Format("20060102150405"))
	attrs := dnsimple.ZoneRecordAttributes{
		Type:    "TXT",
		Name:    &c.Hostname,
		Content: fmt.Sprintf("distribution-check: %s", c.Timing.StartAt.Format(time.RFC3339)),
	}

	c.Timings.Create = NewTiming()
	record, err := client.Zones.CreateRecord(ctx, accountID, domain, attrs)
	c.Timings.Create.Stop()
	if err != nil {
		return &c, err
	}

	c.Created = true

	recordID := record.Data.ID

	// Delete the record when we're done
	defer func() {
		c.Timings.Delete = NewTiming()
		_, err := client.Zones.DeleteRecord(context.WithoutCancel(ctx), accountID, domain, recordID)
		c.Timings.Delete.Stop()
		c.Deleted = err == nil
	}()

	for {
		select {
		case <-ctx.Done():
			return &c, context.Cause(ctx)
		case <-time.After(poll):
			c.Checks++
			t := NewTiming()
			response, err := client.Zones.CheckZoneRecordDistribution(ctx, accountID, domain, recordID)
			c.Timings.Check = append(c.Timings.Check, t.Stop())
			if err != nil {
				return &c, err
			}
			if response.Data.Distributed {
				return &c, nil
			}
		}
	}
}
