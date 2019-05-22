# imsy

## overview

`imsy` shows the underlying principle of file replication
mechanism suitable for large immutable files, such as VM images.

The core idea is to split a file in chunks using a Content Defined Chunking (CDC) mechanism,
and save chunks in a Content Addressed Store (CAS), where each chunk is identified by its hash (e.g. SHA256)

The file can now be fully recovered by knowing the list of hashes of its constituent chunks, in order.

## usage

First check out and run `go build`.

Then get hold of a couple of big files that are different but related, e.g. two VM images. Squashed uncompressed docker images would work too. Let's call them `vm1.img` and `vm2.img`.

Then run:

```
$ imsy -dir server1data prepare <vm1.img
72d21dabc5a57782eaad5745f968a58cd3f029c00897a4da5688e795256dac50
```

This will fill `server1data` with chunks of `vm1.img`.

Now, we want show how to pull this VM from another machine .

On the first machine, run:

```
$ imsy -dir server1data serve
```

On the second machine (it works also on the same machine via localhost):

```
$ imsy -dir server2data pull 72d21dabc5a57782eaad5745f968a58cd3f029c00897a4da5688e795256dac50
```

Since `server2data` on the second machine is empty, this will basically just pull the whole image.
It would have been cheaper to just serve the whole file via HTTP.

But now, on the first machine, you can run:

```
$ imsy -dir server2data prepare <vm2.img
ff62efe2d8f6c4a3b488fafe9ec9046dee5d2fab5b0a5488506bb3af766eacff
```

When you pull that image on the second machine, you'll notice that only a small number of chunks gets actually downloaded (look at the `imsy serve` log)

```
$ imsy -dir server2data pull 72d21dabc5a57782eaad5745f968a58cd3f029c00897a4da5688e795256dac50
```

