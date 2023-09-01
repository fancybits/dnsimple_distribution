# DNSimple DNS Distribution checker

A way to measure the end-to-end latency of record propagation on DNSimple.

## Usage

    $ go build .
    $ ./dnsimple_distribution --token=<oauth-token> --domain=yourdomain.com
    