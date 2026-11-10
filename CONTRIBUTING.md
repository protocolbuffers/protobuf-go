# Contributing to Go Protocol Buffers

Go protocol buffers is an open source project and accepts contributions.
The source of truth for this repository is at
[github.com/protocolbuffers/protobuf-go](https://github.com/protocolbuffers/protobuf-go).
Changes are sent and reviewed as
[GitHub pull requests](https://docs.github.com/en/pull-requests).
The repository at
[go.googlesource.com/protobuf](https://go.googlesource.com/protobuf)
is a mirror, and we no longer accept changes through
[Gerrit](https://go-review.googlesource.com).


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
Below are the steps to send a change as a pull request.

If you are a Googler, it is preferable to first create an internal CL and
have it reviewed and submitted. The code propagation process will deliver
the change to GitHub.


**Step 1:** [Fork](https://github.com/protocolbuffers/protobuf-go/fork)
the repository and clone your fork:
```
$ git clone https://github.com/[your-username]/protobuf-go
```

**Step 2:** Setup a Git hook:
Setup a hook to run the tests prior to pushing changes to GitHub:
```
$ (cd protobuf-go/.git/hooks && echo -e '#!/bin/bash\n./test.bash' > pre-push && chmod a+x pre-push)
```

**Step 3:** Prepare changes in a new branch, created from the `master` branch:

```
$ git checkout -b mybranch
$ [edit files...]
$ git add [files...]
$ git commit
```

**Step 4:** Push the branch to your fork, which will trigger the test hook
that was setup in step 2:
```
$ git push origin mybranch
```

Then
[open a pull request](https://docs.github.com/en/pull-requests/how-tos/create-pull-requests/creating-a-pull-request-from-a-fork)
against the `master` branch of `protocolbuffers/protobuf-go`.
The title and description of the pull request become the commit message,
so please write them like a
[Go commit message](https://go.dev/doc/contribute#commit_messages).

**Step 5:** After a review, there may be changes that are required.
Do so by adding commits to the same branch and pushing it again:
```
$ [edit files...]
$ git add [files...]
$ git commit
$ git push origin mybranch
```

The tests also run on GitHub Actions. For a pull request from a fork,
a maintainer has to start them by adding the `:a: safe for tests` label,
and has to do so again after each push.

Once a maintainer approves the pull request, it is imported into Google's
internal repository, and the resulting commit is then published back to this
repository. Because of this, GitHub may show your pull request as closed
rather than merged.

The [Contribution Guidelines](https://golang.org/doc/contribute.html) for the
Go project provides additional details that are also relevant to
contributing to the Go `protobuf` project, except for the parts about Gerrit.
