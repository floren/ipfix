package ipfix

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
)

type cbval struct {
	eid  uint32
	fid  uint16
	data []byte
}

var (
	walkerPkt     []byte
	totalItems    = (15*14 + 16*1 + 15*1 + 16*2 + 15*2 + 16*1 + 15*4) //total number of items in the packet
	filteredItems = (14*2 + 2 + 2 + 4 + 4 + 2 + 8)                    //only looking at source and dest items
)

func init() {
	//fairly simple ipfix flow record with template in the packet
	walkerPkt, _ = hex.DecodeString("000a05785df00ac2000000d200000000000200440103000f00080004000c0004000f000400070002000b000200060001000a0002000e000200020004000100040098000800990008000400010005000100880001010302a47f0000017f00000100000000b59f080700ffff000100000004000012080000016ef1a9ae8d0000016ef1a9aef51100017f0000017f00000100000000b59f0807000001ffff00000004000012080000016ef1a9ae8d0000016ef1a9aef5110001c0a87a01c0a87aff00000000445c445c000003ffff000000010000009e0000016ef1a9b2a00000016ef1a9b2a0110001ac110001ac11ffff00000000445c445c00ffff0007000000010000009e0000016ef1a9b2a00000016ef1a9b2a0110001ac110001ac11ffff00000000445c445c000007ffff000000010000009e0000016ef1a9b2a00000016ef1a9b2a0110001ac130001ac1300ff00000000445c445c00ffff0006000000010000009e0000016ef1a9b2a00000016ef1a9b2a01100010a000064ffffffff00000000445c445c000002ffff000000010000009e0000016ef1a9b2a00000016ef1a9b2a0110001c0a87a01c0a87aff00000000445c445c00ffff0003000000010000009e0000016ef1a9b2a00000016ef1a9b2a01100010a000064ffffffff00000000445c445c00ffff0002000000010000009e0000016ef1a9b2a00000016ef1a9b2a0110001ac120001ac12ffff00000000445c445c00ffff0005000000010000009e0000016ef1a9b2a00000016ef1a9b2a01100010a0000640a0000ff00000000445c445c000002ffff000000010000009e0000016ef1a9b2a00000016ef1a9b2a0110001ac130001ac1300ff00000000445c445c000006ffff000000010000009e0000016ef1a9b2a00000016ef1a9b2a01100010a0000640a0000ff00000000445c445c00ffff0002000000010000009e0000016ef1a9b2a00000016ef1a9b2a0110001ac120001ac12ffff00000000445c445c000005ffff000000010000009e0000016ef1a9b2a00000016ef1a9b2a0110001000200480104001000080004000c0004000f000400070002000b000200060001000a0002000e00020002000400010004009800080099000800040001000500010088000100d100040104003803d372530a0000640000000001bbda5a180002ffff000000010000006c0000016ef1a9bf250000016ef1a9bf2506000181000000010300344a7d8ebd0a0000640000000001bb9d2c000002ffff00000002000002320000016ef1a9b1000000016ef1a9c8b81100010104006cc1b609730a0000640000000001bbdeb0100002ffff00000001000000340000016ef1a9c9000000016ef1a9c900060001810000000a000064c1b609730a000001deb001bb10ffff000200000001000000340000016ef1a9c8e40000016ef1a9c8e406000181000000010300640a000064010101010a0000018404003500ffff000200000001000000480000016ef1a9cbe00000016ef1a9cbe01100017f0000017f00003500000000ce790035000001ffff000000010000003d0000016ef1a9cbe00000016ef1a9cbe0110001010400380a000064c01eff750a000001e63801bb14ffff000200000001000000340000016ef1a9cbe00000016ef1a9cbe006000381000000010300c47f0000017f00003500000000ce79003500ffff0001000000010000003d0000016ef1a9cbe00000016ef1a9cbe01100017f0000357f000001000000000035ce7900ffff000100000001000000680000016ef1a9cc080000016ef1a9cc081100017f0000357f000001000000000035ce79000001ffff00000001000000680000016ef1a9cc080000016ef1a9cc08110001010101010a0000640000000000358404000002ffff00000001000000730000016ef1a9cc080000016ef1a9cc08110001")

}

func TestIPFixWalk(t *testing.T) {
	var f Filter
	f.SetVersion(10)
	f.SetDomainID(0)

	//test the first couple flows in the packet
	testSet := []cbval{
		cbval{fid: 8, data: []byte{0x7f, 0x00, 0x00, 0x01}},
		cbval{fid: 12, data: []byte{0x7f, 0x00, 0x00, 0x01}},
		cbval{fid: 15, data: []byte{0x00, 0x00, 0x00, 0x00}},
		cbval{fid: 7, data: []byte{0xb5, 0x9f}},
		cbval{fid: 11, data: []byte{0x08, 0x07}},
		cbval{fid: 6, data: []byte{0x00}},
	}

	var cnt int
	cb := func(mh MessageHeader, eid uint32, fid uint16, buff []byte) error {
		if eid != 0 {
			return errors.New("invalid enterprise id")
		}
		ft, ok := IPfixIDTypeLookup(eid, fid)
		if !ok {
			return fmt.Errorf("Unknown type %d %d", eid, fid)
		}
		if len(buff) < ft.minLength() {
			return fmt.Errorf("Returned buffer is too small for type: %d < %d", len(buff), ft.minLength())
		}
		if cnt < len(testSet) {
			if eid != testSet[cnt].eid {
				return fmt.Errorf("Flow %d EID bad: %d != %d", cnt, eid, testSet[cnt].eid)
			}
			if fid != testSet[cnt].fid {
				return fmt.Errorf("Flow %d EID bad: %d != %d", cnt, fid, testSet[cnt].fid)
			}
			if !bytes.Equal(buff, testSet[cnt].data) {
				return fmt.Errorf("Bad data: %v != %v", buff, testSet[cnt].data)
			}
		}
		cnt++
		return nil
	}

	w, err := NewWalker(&f, cb, 16, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.WalkBuffer(walkerPkt); err != nil {
		t.Fatal(err)
	}
	//check the number of call backs against what is in the packet
	if cnt != totalItems {
		t.Fatalf("invalid count: %d != %d", cnt, totalItems)
	}
}

func TestIPFixWalkFilter(t *testing.T) {
	var f Filter
	f.SetVersion(10)
	f.SetDomainID(0)
	// ONLY want SrcAddr and DstAddr
	f.Set(0, 0x8)
	f.Set(0, 12)

	var cnt int
	cb := func(mh MessageHeader, eid uint32, fid uint16, buff []byte) error {
		if eid != 0 || !(fid == 0x8 || fid == 12) {
			return errors.New("invalid filtered set")
		} else if len(buff) != 4 {
			//IPv4 address
			return errors.New("Invalid data size")
		}
		cnt++
		return nil
	}
	w, err := NewWalker(&f, cb, 16, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err = w.WalkBuffer(walkerPkt); err != nil {
		t.Fatal(err)
	}
	if cnt != filteredItems {
		t.Fatalf("Bad item count with filter: %d != %d", cnt, filteredItems)
	}
}

func BenchmarkFullWalk(b *testing.B) {
	var cnt int
	cb := func(mh MessageHeader, eid uint32, fid uint16, buff []byte) error {
		if eid != 0 {
			return errors.New("invalid enterprise id")
		}
		cnt++
		return nil
	}
	w, err := NewWalker(nil, cb, 16, 1024)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cnt = 0
		if err = w.WalkBuffer(walkerPkt); err != nil {
			b.Fatal(err)
		}
		if cnt != totalItems {
			b.Fatalf("Bad item count: %d != %d", cnt, totalItems)
		}
	}
	b.SetBytes(int64(b.N * len(walkerPkt)))
}

func BenchmarkFilterWalk(b *testing.B) {
	var f Filter
	f.SetVersion(10)
	f.Set(0, 8)
	f.Set(0, 12)
	var cnt int
	cb := func(mh MessageHeader, eid uint32, fid uint16, buff []byte) error {
		if eid != 0 {
			return errors.New("invalid enterprise id")
		} else if len(buff) != 4 {
			//IPv4 address
			return errors.New("Invalid data size")
		}
		cnt++
		return nil
	}
	w, err := NewWalker(&f, cb, 16, 1024)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cnt = 0
		if err = w.WalkBuffer(walkerPkt); err != nil {
			b.Fatal(err)
		}
		if cnt != filteredItems {
			b.Fatalf("Bad item count: %d != %d", cnt, filteredItems)
		}
	}
	b.SetBytes(int64(b.N * len(walkerPkt)))
}
