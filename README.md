# Reed-Solomon File Protector

[![Build Status](https://travis-ci.com/AlexHalogen/RSFileProtect.svg?branch=main)](https://travis-ci.com/AlexHalogen/RSFileProtect)

A simple tool for file protection based on Reed-Solomon coding.

**CURRENTLY UNDER DEVELOPMENT!!!**

## Features

- Separate ecc file(only two!) from original data
- Fast verification based on crc hashes

## Suitable for...

- Detecting and repairing in-place bit rots

## Drawbacks

- Less resistant to larger damages to disks

## TODO

- [ ] More friendly command-line interface
- [ ] Implement finer decoding algorithm for higher chances for successful repairs(Berlekamp-Massey, etc)
- [ ] Custom ecc symbol ratio
- [ ] Custom chunk size
- [ ] Optimized strategy for small files

## Special Thanks to 

This tool depends on the wonderful [reedsolomon library](https://github.com/klauspost/reedsolomon) written by Klaus Post(@klauspost). The reedsolomon library was released under MIT license. See source file in encoding/decoding package for detailed license info.
