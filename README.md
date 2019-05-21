= imsy

== overview

`imsy` shows the underlying principle of file replication
mechanism suitable for large immutable files, such as VM images.

The core idea is to split a file in chunks using a Content Defined Chunking (CDC) mechanism,
and save chunks in a Content Addressed Store (CAS), where each chunk is identified by its hash (e.g. SHA256)

The file can now be fully recovered by knowing the list of hashes of its constituent chunks, in order.

== usage

1. check out and run `go build`

2. get hold of a couple of big files that are different but related, e.g. two VM images. Squashed uncompressed docker images would work too.

2. `imsy prepare <file-v1 >file-v1.hashes`

3. `