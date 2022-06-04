# lightsshd

A lightweight SSH daemon

## Introduction

### Why?

My Kindle is running a super outdated version of Dropbear, and I hate recompiling stuff in C.
For this reason, this small SSH Server port will allow me to have a customized SSH daemon that I can
tweak as needed.

Go supports easy cross-compilation (via `GOOS=linux GOARCH=armv7` for example) and statically linked binaries (`CGO_ENABLED=0`),
making it the perfect language for my binaries.

### Not supported features

- User switching (`ssh root@127.0.0.1 -p 2222` will results in a `$USER` shell)
- SFTP (Not tested)
- Non-ed25519 host keys

## Getting Started

```
$ make build
$ ./build/lightsshd -h
Usage: lightsshd [--loglevel LOGLEVEL] [--address ADDRESS] [--hostkeyfile HOSTKEYFILE] [--authorizedkeys AUTHORIZEDKEYS]

Options:
  --loglevel LOGLEVEL, -l LOGLEVEL
  --address ADDRESS, -L ADDRESS [default: 0.0.0.0:2222]
  --hostkeyfile HOSTKEYFILE, -k HOSTKEYFILE [default: /etc/lightsshd/ssh_host_ed25519_key]
  --authorizedkeys AUTHORIZEDKEYS, -a AUTHORIZEDKEYS [default: /etc/lightsshd/authorized_keys]
  --help, -h             display this help and exit
$ sudo mkdir /etc/lightsshd/ && sudo chown $(id -u):$(id -g) /etc/lightsshd/
$ ssh-add -L > /etc/lightsshd/authorized_keys
$ ./build/lightsshd -l debug -L 127.0.0.1:2222 &
$ ssh anything@127.0.0.1 -p 2222 id
uid=1000(user) gid=1000(user) groups=1000(user)
```