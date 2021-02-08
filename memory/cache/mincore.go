package cache

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"unsafe"
)

// mmap the given file, get the mincore vector, then
// return it as an []bool
func FileMincore(f *os.File, size int64) ([]bool, error) {
	//skip could not mmap error when the file size is 0
	if int(size) == 0 {
		return nil, nil
	}
	// mmap is a []byte
	mmap, err := unix.Mmap(int(f.Fd()), 0, int(size), unix.PROT_NONE, unix.MAP_SHARED)

	if err != nil {
		return nil, fmt.Errorf("could not mmap: %v", err)
	}
	// TODO: check for MAP_FAILED which is ((void *) -1)
	// but maybe unnecessary since it looks like errno is always set when MAP_FAILED

	// one byte per page, only LSB is used, remainder is reserved and clear
	vecsz := (size + int64(os.Getpagesize()) - 1) / int64(os.Getpagesize())
	vec := make([]byte, vecsz)

	// get all of the arguments to the mincore syscall converted to uintptr
	mmap_ptr := uintptr(unsafe.Pointer(&mmap[0]))
	size_ptr := uintptr(size)
	vec_ptr := uintptr(unsafe.Pointer(&vec[0]))

	// use Go's ASM to submit directly to the kernel, no C wrapper needed
	// mincore(2): int mincore(void *addr, size_t length, unsigned char *vec);
	// 0 on success, takes the pointer to the mmap, a size, which is the
	// size that came from f.Stat(), and the vector, which is a pointer
	// to the memory behind an []byte
	// this writes a snapshot of the data into vec which a list of 8-bit flags
	// with the LSB set if the page in that position is currently in VFS cache
	ret, _, err := unix.Syscall(unix.SYS_MINCORE, mmap_ptr, size_ptr, vec_ptr)
	if ret != 0 {
		return nil, fmt.Errorf("syscall SYS_MINCORE failed: %v", err)
	}
	defer unix.Munmap(mmap)

	mc := make([]bool, vecsz)

	// there is no bitshift only bool
	for i, b := range vec {
		if b%2 == 1 {
			mc[i] = true
		} else {
			mc[i] = false
		}
	}
	return mc, nil
}
