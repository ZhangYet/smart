// Copyright 2017 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// NVMe admin commands.

package smart

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	NVME_ADMIN_IDENTIFY = 0x06
)

var (
	NVME_IOCTL_ADMIN_CMD = _iowr('N', 0x41, unsafe.Sizeof(nvmePassthruCommand{}))
)

// Defined in <linux/nvme_ioctl.h>
type nvmePassthruCommand struct {
	opcode       uint8
	flags        uint8
	rsvd1        uint16
	nsid         uint32
	cdw2         uint32
	cdw3         uint32
	metadata     uint64
	addr         uint64
	metadata_len uint32
	data_len     uint32
	cdw10        uint32
	cdw11        uint32
	cdw12        uint32
	cdw13        uint32
	cdw14        uint32
	cdw15        uint32
	timeout_ms   uint32
	result       uint32
} // 72 bytes

type nvmeIdentPowerState struct {
	MaxPower        uint16 // Centiwatts
	Rsvd2           uint8
	Flags           uint8
	EntryLat        uint32 // Microseconds
	ExitLat         uint32 // Microseconds
	ReadTput        uint8
	ReadLat         uint8
	WriteTput       uint8
	WriteLat        uint8
	IdlePower       uint16
	IdleScale       uint8
	Rsvd19          uint8
	ActivePower     uint16
	ActiveWorkScale uint8
	Rsvd23          [9]byte
}

type nvmeIdentController struct {
	VendorID     uint16
	Ssvid        uint16
	SerialNumber [20]byte
	ModelNumber  [40]byte
	Firmware     [8]byte
	Rab          uint8
	IEEE         [3]byte
	Cmic         uint8
	Mdts         uint8
	Cntlid       uint16
	Ver          uint32
	Rtd3r        uint32
	Rtd3e        uint32
	Oaes         uint32
	Rsvd96       [160]byte
	Oacs         uint16
	Acl          uint8
	Aerl         uint8
	Frmw         uint8
	Lpa          uint8
	Elpe         uint8
	Npss         uint8
	Avscc        uint8
	Apsta        uint8
	Wctemp       uint16
	Cctemp       uint16
	Mtfa         uint16
	Hmpre        uint32
	Hmmin        uint32
	Tnvmcap      [16]byte
	Unvmcap      [16]byte
	Rpmbs        uint32
	Rsvd316      [196]byte
	Sqes         uint8
	Cqes         uint8
	Rsvd514      [2]byte
	Nn           uint32
	Oncs         uint16
	Fuses        uint16
	Fna          uint8
	Vwc          uint8
	Awun         uint16
	Awupf        uint16
	Nvscc        uint8
	Rsvd531      uint8
	Acwu         uint16
	Rsvd534      [2]byte
	Sgls         uint32
	Rsvd540      [1508]byte
	Psd          [32]nvmeIdentPowerState
	Vs           [1024]byte
}

// WIP, highly likely to change
func OpenNVMe(dev string) error {
	fd, err := syscall.Open(dev, syscall.O_RDWR, 0600)
	if err != nil {
		return err
	}

	defer syscall.Close(fd)

	buf := make([]byte, 4096)

	cmd := nvmePassthruCommand{
		opcode:   NVME_ADMIN_IDENTIFY,
		nsid:     0, // Namespace 0, since we are identifying the controller
		addr:     uint64(uintptr(unsafe.Pointer(&buf[0]))),
		data_len: uint32(len(buf)),
		cdw10:    1, // Identify controller
	}

	fmt.Printf("unsafe.Sizeof(cmd): %d\n", unsafe.Sizeof(cmd))
	fmt.Printf("binary.Size(cmd): %d\n", binary.Size(cmd))

	if err := ioctl(uintptr(fd), NVME_IOCTL_ADMIN_CMD, uintptr(unsafe.Pointer(&cmd))); err != nil {
		return err
	}

	fmt.Printf("NVMe call: opcode=%#02x, size=%#04x, nsid=%#08x, cdw10=%#08x\n",
		cmd.opcode, cmd.data_len, cmd.nsid, cmd.cdw10)

	var controller nvmeIdentController

	// Should be 4096
	fmt.Printf("binary.Size(controller): %d\n", binary.Size(controller))

	binary.Read(bytes.NewBuffer(buf[:]), nativeEndian, &controller)

	fmt.Printf("%+v\n", controller)
	fmt.Println()
	fmt.Printf("Vendor ID: %#04x\n", controller.VendorID)
	fmt.Printf("Model number: %s\n", controller.ModelNumber)
	fmt.Printf("Serial number: %s\n", controller.SerialNumber)
	fmt.Printf("Firmware version: %s\n", controller.Firmware)
	fmt.Printf("IEEE OUI identifier: 0x%02x%02x%02x\n",
		controller.IEEE[2], controller.IEEE[1], controller.IEEE[0])

	return nil
}
