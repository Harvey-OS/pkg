/*
Copyright 2018 Harvey OS Team

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice,
   this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors
   may be used to endorse or promote products derived from this software
   without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO,
THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR
CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS;
OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR
OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF
ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package main

import (
	"debug/elf"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
)

const LRES = 3

var kernel = flag.String("k", "9k", "kernel name")

func main() {
	flag.Parse()

	n := flag.Args()[0]
	d, err := os.Open(n)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	f, err := elf.Open(*kernel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	var codeend uint64
	var codestart uint64 = math.MaxUint64

	for _, v := range f.Progs {
		if v.Type != elf.PT_LOAD {
			continue
		}
		fmt.Fprintf(os.Stderr, "processing %v\n", v)
		// MUST alignt to 2M page boundary.
		// then MUST allocate a []byte that
		// is the right size. And MUST
		// see if by some off chance it
		// joins to a pre-existing segment.
		// It's easier than it seems. We produce ONE text
		// array and ONE data array. So it's a matter of creating
		// a virtual memory space with an assumed starting point of
		// 0x200000, and filling it. We just grow that as needed.

		curstart := v.Vaddr
		curend := v.Vaddr + v.Memsz
		// magic numbers, BAH!
		if curstart < uint64(0xffffffff00000000) {
			curstart += 0xfffffffff0000000
			curend += 0xfffffffff0000000
		}
		fmt.Fprintf(os.Stderr, "s %x e %x\n", curstart, curend)
		if v.Flags&elf.PF_X == elf.PF_X {
			if curstart < codestart {
				codestart = curstart
			}
			if curend > codeend {
				codeend = curend
			}
			fmt.Fprintf(os.Stderr, "code s %x e %x\n", codestart, codeend)
		}
	}
	fmt.Fprintf(os.Stderr, "code s %x e %x\n", codestart, codeend)
	s, err := f.Symbols()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	// maybe we should stop doing LRES ...
	symname := make([]string, codeend-codestart)
	for i := range symname {
		symname[i] = fmt.Sprintf("[0x%x]", codestart+uint64(i))
	}
	for _, v := range s {
		vstart := v.Value
		vend := v.Value + v.Size
		if v.Value > codeend {
			continue
		}
		if v.Value+v.Size < codestart {
			continue
		}
		if vstart < codestart {
			v.Value = codestart
		}
		if vend > codeend {
			vend = codeend
		}
		for i := vstart; i < vend; i++ {
			symname[i-codestart] = v.Name
		}
	}
	symname[0] = "Total ms"
	symname[1<<LRES] = "Unknown"
	// now dump the info ...
	count := uint32(0)
	pc := codestart
	for {
		if err := binary.Read(d, binary.BigEndian, &count); err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			break
		}
		if count > 0 {
			fmt.Printf("%s %d\n", symname[pc-codestart], count)
		}
		pc += (1 << LRES)
	}
}
