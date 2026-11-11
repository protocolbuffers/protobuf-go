# Contributing to Go Protocol Buffers

Go protocol buffers is an open source project and accepts contributions.
The source of truth for this repository is at
[github.com/protocolbuffers/protobuf-go](https://github.com/protocolbuffers/protobuf-go).
Changes are sent and reviewed as
[GitHub pull requests](https://docs.github.com/en/pull-requests).

The repository at
[go.googlesource.com/protobuf](https://go.googlesource.com/protobuf)
was the source of truth until November 2026 and is now deprecated.
It is only a mirror of the GitHub repository, and we no longer accept
changes through [Gerrit](https://go-review.googlesource.com).


## Becoming a contributor

Contributions to this project must be accompanied by a
[Contributor License Agreement](https://cla.developers.google.com/about) (CLA).
You (or your employer) retain the copyright to your contribution;
the CLA gives us permission to use and redistribute your contribution
as part of the project.
If you or your employer have already signed the Google CLA,
you do not need to sign it again.
Visit <https://cla.developers.google.com/> to see your current agreements
or to sign a new one.

The project's tests also require Bazel. See http://bazel.build/install for
installation instructions.


## Before contributing code

The project welcomes submissions, but to make sure things are well coordinated
we ask that contributors discuss any significant changes before starting work.
Best practice is to connect your work to the
[issue tracker](https://github.com/golang/protobuf/issues),
either by filing a new issue or by claiming an existing issue.


## Sending a change via GitHub

The `protobuf` project performs development on GitHub.
Changes are sent as pull requests against the `master` branch.
If you are new to pull requests, see
[GitHub's documentation](https://docs.github.com/en/pull-requests/reference/pull-requests).

If you are a Googler, it is preferable to first create an internal CL and
have it reviewed and submitted. The code propagation process will deliver
the change to GitHub.

The title and description of the pull request become the commit message,
so please write them like a
[Go commit message](https://go.dev/doc/contribute#commit_messages).

Before sending a pull request, please run the tests
(see also [Test your changes](https://go.dev/doc/contribute#testing)
in the Go contribution guide):
```
$ ./test.bash
```

The tests also run on GitHub Actions. For a pull request from a fork,
a maintainer has to start them by adding the `:a: safe for tests` label,
and has to do so again after each push.
