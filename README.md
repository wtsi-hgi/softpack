# Softpack

[![Go](https://github.com/wtsi-hgi/softpack/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/wtsi-hgi/softpack/actions/workflows/go.yml)

Softpack backend service.

Starts a web server with various API endpoints allowing users to create and manage 
environments and request packages. Builds environments into singularity containers
using binaries from the configured apt package repository.
--

## Usage

Create a config file config.yaml of the following format:

```
base-img-path: Local path or docker:// URL for the base installation image
temp-dir: Local directory used to extract and modify the image. If unprovided,
    one will be created inside of the install-dir.
install-dir: Local directory where the final container image and generated
    symlinks will be stored.
wrapper-script: Local path to the wrapper script that generated symlinks
    will point to
apt-src: Local directory, HTTP URL, or s3:// URL pointing to the APT
    repository used to install packages into the image.
module-path: Local directory where generated module files will be placed.
artefact-store: Local path or s3:// URL used to store generated artefacts.
db-conn: Database connection string, either a local SQLite path or a
    mysql:// style URI for a remote database.
listen-addr: Address where the web server will listen.
smtp: Open SMTP server address (Optional)
email-domain: Email domain Eg. @example.com. Build status emails to admin will be
    sent from user<email-domain>. (Optional)
admin: Admin email address for build status emails to be sent/recieved from. (Optional)
```

The service can then be run with:

```
go run . server --config /path/to/config.yaml
```

To run all tests without skips install:

```
sudo apt-get update
sudo apt-get install -y singularity-container squashfs-tools
```

Then run tests:

```
go test ./...
```