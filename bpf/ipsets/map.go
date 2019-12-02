// Copyright (c) 2019 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ipsets

import (
	"encoding/binary"
	"net"

	"golang.org/x/sys/unix"

	"github.com/projectcalico/felix/bpf"
	"github.com/projectcalico/felix/ip"
)

// uint32 prefixLen HE  4
// uint64 set_id BE     +8 = 12
// uint32 addr BE       +4 = 16
// uint16 port HE       +2 = 18
// uint8 proto          +1 = 19
// uint8 pad            +1 = 20
const ipSetEntrySize = 20

type IPSetEntry [ipSetEntrySize]byte

func Map() bpf.Map {
	return bpf.NewPinnedMap(bpf.MapParameters{
		Filename:   "/sys/fs/bpf/tc/globals/calico_ip_sets",
		Type:       "lpm_trie",
		KeySize:    ipSetEntrySize,
		ValueSize:  4,
		MaxEntries: 1024 * 1024,
		Name:       "calico_ip_sets",
		Flags:      unix.BPF_F_NO_PREALLOC,
	})
}

func (e IPSetEntry) SetID() uint64 {
	return binary.BigEndian.Uint64(e[4:12])
}

func (e IPSetEntry) Addr() net.IP {
	return e[12:16]
}

func (e IPSetEntry) PrefixLen() uint32 {
	return binary.LittleEndian.Uint32(e[:4])
}

func (e IPSetEntry) Protocol() uint8 {
	return e[18]
}

func (e IPSetEntry) Port() uint16 {
	return binary.LittleEndian.Uint16(e[16:18])
}

func MakeBPFIPSetEntry(setID uint64, cidr ip.V4CIDR, port uint16, proto uint8) IPSetEntry {
	var entry IPSetEntry
	// TODO Detect endianness
	if proto == 0 {
		// Normal CIDR-based lookup.
		binary.LittleEndian.PutUint32(entry[0:4], uint32(64 /* ID */ +cidr.Prefix()))
	} else {
		// Named port lookup, use full length of key.
		binary.LittleEndian.PutUint32(entry[0:4], 64 /* ID */ +32 /* IP */ +16 /* Port */ +8 /* protocol */)
	}
	binary.BigEndian.PutUint64(entry[4:12], setID)
	binary.BigEndian.PutUint32(entry[12:16], cidr.Addr().(ip.V4Addr).AsUint32())
	binary.LittleEndian.PutUint16(entry[16:18], port)
	entry[18] = proto
	return entry
}

var DummyValue = []byte{1, 0, 0, 0}
