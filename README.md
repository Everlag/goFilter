goFilter
========

A basic bloom filter implementation in the wonderful go. This is the destroyer of caches.

Uint64s are used via bitwise operations to get 512MB of memory usage versus the 4GB that would be used by a naive bool array implementation.

SHA256 is the supported hash function as my uses required a cryptographic quality hash. Switching for a faster hash function from the standard lib would be trivial.

Serialization is supported to JSON with optional compression.

Testing and Benching
------
A test file is included. Run `go test` while in the directory the package is in and you'll get if it passes.
The serialization tests are commented out for the fact that they take a long time.


Benchmarks are run under the same environment as the tests but with `go test -bench=".*"`

Three degrees of usage are given for benchmarks which are based around hash iterations per item address lookup. Additionally, a AddSpeedStandardHashRef is provided to gauge the performance difference between a naive implementation relying on a bool array verus the provided bitwise method.

Sample benchmark results from an i5 2500k with a terrible 16GB memory configuration(Two different vendors and base clock speeds.... ewww) is:

    BenchmarkAddSpeedLargeHash           100          18487347 ns/op
    BenchmarkAddSpeedModerateHash       1000           1818730 ns/op
    BenchmarkAddSpeedStandardHash     100000             24468 ns/op
    BenchmarkAddSpeedStandardHashRef          100000             19687 ns/op
    BenchmarkCheckSpeedLargeHash         100          12951644 ns/op
    BenchmarkCheckSpeedModerateHash     2000           1293164 ns/op
    BenchmarkCheckSpeedStandardHash   100000             19482 ns/op