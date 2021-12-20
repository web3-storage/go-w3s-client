# w3s CLI

```
Upload some files or a directories to web3.storage.
It deals with spliting CAR archives into many 100Mib CARs.

Environment variables:
WEB3_STORAGE_KEY = <web3.storage api key>

Positionals:
<file or directory to add>

Flags:
  -car
    	Upload a CAR archive as is rather than as a file.
  -help
    	Shows the help message.
  -no-stdin
    	Disable understanding the "-" path as /dev/stdin and understand it as a file named -.
  -quiet
    	Only outputs the resulting hash (script friendly option).

Example:
WEB3_STORAGE_KEY="eyJhbGc...TSyYE" ./w3s someFiles/
```

## Install

```bash
go install github.com/web3-storage/go-w3s-client/cli/w3s@latest
```
