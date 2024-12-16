# semver

The `semver` package is a wrapper around the `golang.org/x/mod/semver` package. It provides the same functions as the original package plus a few additional ones that may be useful, allowing you to work consistently with semantic versioning in Go.

## Example Usage

```
import (
    "github.com/phil-inc/pcommon/semver"
)

VERSION := v1.0.0

valid := semver.IsValid(VERSION)
majorVersion := semver.Major(VERSION)
release := semver.Increment(VERSION, semver.MAJOR_VERSION_INCREMENT)
```