# statsite-instrumental

[![Build Status](https://travis-ci.org/collectiveidea/statsite-instrumental.png?branch=master)](https://travis-ci.org/collectiveidea/statsite-instrumental)

This project is a sink for [statsite](https://github.com/armon/statsite)
and [Instrumental](https://instrumentalapp.com) written in Go.

## Statsite Config

```ini
[statsite]
stream_cmd = statsite-instrumental this_is_your_token
flush_interval = 60
```

It is important to set the `flush_interval` to 60s as that is the highest
resolution supported by Instrumental. If the default of 10 seconds is kept then
only the last value flushed will be stored. This can cause misleading metrics
as it will not reflect the correct percentiles, uppers, lowers, etc...

## Usage

`statsite-instrumental` requires the API token for your project be passed as
the first argument. The API token can be found on the Settings page for your
project.

### --prefix

A string that is added at the beginning of tags sent to Instrumental. The prefix
is added as given and it's good practice to add a `.` at the end.

Example: `--prefix 'production.'`

### --postfix

A string that is added at the end of each tag sent to Instrumental. It's good
practice to start the postfix with a `.`.

Example: `` --prefix `hostname` ``

### --timeout

The timeout for talking to Instrumental. Support a couple different formats.

Examples:
- `1m`
- `30s`
- `1h20m5s`

### --host

The Instrumental collector agent hostname. You probably won't want to actually
change this.

### --port

The Instrumental collector agent port to connect on. Again, unlikely to have to
change this.
