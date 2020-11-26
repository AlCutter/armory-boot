// https://github.com/f-secure-foundry/armory-boot
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"debug/elf"
	"fmt"

	"github.com/f-secure-foundry/tamago/arm"
	"github.com/f-secure-foundry/tamago/board/f-secure/usbarmory/mark-two"
	"github.com/f-secure-foundry/tamago/dma"
	"github.com/f-secure-foundry/tamago/soc/imx6"
)

// bootELFUnikernel attempts to load the provided ELF image and jumps to it.
//
// This function implements a _very_ simple ELF loader which is suitable for
// loading bare-metal ELF files like those produced by Tamago.
func bootELFUnikernel(img []byte) {
	dma.Init(dmaStart, dmaSize)
	mem, _ := dma.Reserve(dmaSize, 0)

	f, err := elf.NewFile(bytes.NewReader(img))
	if err != nil {
		panic(err.Error)
	}

	for idx, prg := range f.Progs {
		if prg.Type == elf.PT_LOAD {
			b := make([]byte, prg.Memsz)
			_, err := prg.ReadAt(b[0:prg.Filesz], 0)
			if err != nil {
				panic(fmt.Sprintf("Failed to read LOAD section at idx %d: %q", idx, err))
			}
			offset := uint32(prg.Paddr) - mem
			dma.Write(mem, b, int(offset))
		}
	}

	entry := f.Entry

	arm.ExceptionHandler(func(n int) {
		if n != arm.SUPERVISOR {
			panic("unhandled exception")
		}

		fmt.Printf("armory-boot: starting unikernel elf image@%x\n", entry)

		usbarmory.LED("blue", false)
		usbarmory.LED("white", false)

		// TODO(al): There's some issue around the hw rng at the moment:
		//           RNGB.Init() will hang waiting for the RNG to finish
		//           reseeding.
		// imx6.RNGB.Reset()

		imx6.ARM.InterruptsDisable()
		imx6.ARM.CacheFlushData()
		imx6.ARM.CacheDisable()

		// We can re-use the kernel exec even though we don't really need
		// some of the functionality in there.
		exec(uint32(entry), 0)
	})

	svc()
}
