
<p align="center">
  <img width=150 height=150 src="https://github.com/macaroni-os/macaroni-site/blob/master/site/static/images/logo.png">
</p>

# luet-geaaru (fork) - Container-based Package manager

[![codecov](https://codecov.io/gh/geaaru/luet/branch/geaaru/graph/badge.svg?token=LR1IGZKB9X)](https://codecov.io/gh/geaaru/luet)
[![Github All Releases](https://img.shields.io/github/downloads/geaaru/luet/total.svg)](https://github.com/geaaru/luet/releases)

Luet is a multi-platform Package Manager based off from containers - it uses Docker (and others) to build packages. It has zero dependencies and it is well suitable for "from scratch" environments. It can also version entire rootfs and enables delivery of OTA-alike updates, making it a perfect fit for the Edge computing era and IoT embedded devices.

It offers a simple [specfile format](https://luet-lab.github.io/docs/docs/concepts/packages/specfile/) in YAML notation to define both [packages](https://luet-lab.github.io/docs/docs/concepts/packages/) and [rootfs](https://luet-lab.github.io/docs/docs/concepts/packages/#package-layers). As it is based on containers, it can be also used to build stages for Linux From Scratch installations and it can build and track updates for those systems.

It is written entirely in Golang and where used as package manager, it can run in from scratch environment, with zero dependencies.

## Differences between upstream release

There are so many differences that it becomes challenging to create a list.
This is also the reason because probably in the near future my fork will be renamed and rebooted.
This fork has the primary scope to be used in [Macaroni OS](https://www.macaronios.org) with specific
requirements needed for a good integration with [Funtoo Linux](https://funtoo.org) too.

For now, I will try to describe what are the major differences:

* the binary is been splitted, `luet` is the client installer normally used by users
  to install/remove packages, `luet-build` instead is used for build packages,
  create repos, etc.

* there is only one solver now that doesn't use SAT or QLearning.
  It's been totally rewritten.

* i begin to rewrite the docker backend and use the [tar-formers](https://github.com/geaaru/tar-formers/)
  to manage the tar streams. The same library is now used to unpack the tarball and install packages.
  On installation, it's important to ensure that the unpacked files will be synced to the filesystem
  and so the tar-formers library forces a flush and sync to the filesystem,
  this decreases the installation speed. To speed up this there is a section on luet config that
  could be tuned based on the target system:

  ```yaml
  # ---------------------------------------------
  # Tarball flows configuration section:
  # ---------------------------------------------
  tar_flows:
  #
  #   Enable mutex for parallel creation of directories
  #   in the untar specs. Normally this field must be
  #   set to true, the default value.
  #   mutex4dir: true
  #
  #   Define the max number of open files related
  #   to a single untar operation. Be carefour on
  #   set this option with a big value to avoid
  #   'too open files' errors.
  #   In a normal system this could be also 512.
    max_openfiles: 100
  #
  #   Define the buffer size in KB to use
  #   on create files from tar content.
    copy_buffer_size: 32
  ```
  FWIS, increasing these values to 200/300 max open files and using a buffer of 128 could
  improve performance but this depends on disk speed, hardware, RAM, etc.

* `subsets` feature: Permit to define subsets to choice what files extract from original package.
  This means that we could avoid splitting of a package, for example for Portage metadata, Include files, etc.
  and customize the subsets defined in the original package definition with custom options that could be
  configured from an user at runtime.

* `annotations`: the annotations are managed as interface{} struct without define only strings

* drop `extensions` and `plugin` support not used

* drop `reclaim` command.

## Install

To install luet, you can grab a release on the [Release page](https://github.com/geaaru/luet/releases) or to install it in your system:

```bash
$> curl https://raw.githubusercontent.com/geaaru/luet/geaaru/contrib/config/get_luet_root.sh | sh
$> luet search ...
$> luet install ..
$> luet --help
```

## Build from source

```bash
$ git clone https://github.com/geaaru/luet.git -b geaaru
$ cd luet
$ make build
```

## Documentation

Will be soon integrated with Macaroni OS documentation.

## License

Luet is distributed under the terms of GPLv3, check out the LICENSE file.
