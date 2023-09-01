# DNSimple DNS Distribution checker

A way to measure the end-to-end latency of record propagation on DNSimple.

## Usage

    $ go build .
    $ ./dnsimple_distribution --token=<oauth-token> --domain=yourdomain.com
    msg="Sleeping until next interval" delay=39.399069s
    at=2023-08-31T23:09:00-07:00 duration=3.346s checks=1 deleted=true
    at=2023-08-31T23:10:00-07:00 duration=3.364s checks=1 deleted=true
    at=2023-08-31T23:11:00-07:00 duration=3.34s checks=1 deleted=true
    at=2023-08-31T23:12:00-07:00 duration=3.322s checks=1 deleted=true
    at=2023-08-31T23:13:00-07:00 duration=3.391s checks=1 deleted=true
