package pclnpost

import "encoding/binary"

// SymbolEntryCarrierMagic identifies reserved storage whose entries must be
// collected from the final native symbol table, without references in IR.
const SymbolEntryCarrierMagic uint64 = 0x314d59534f474c4c // LLGOSYM1

func symbolEntryCarrier(info *binaryInfo) bool {
	for off := 0; off <= 64 && off+16 <= len(info.entrySec); off += 16 {
		if binary.LittleEndian.Uint64(info.entrySec[off:]) == SymbolEntryCarrierMagic {
			return true
		}
	}
	return false
}

func entryRecords(path string, info *binaryInfo) ([]siteRecord, error) {
	if !symbolEntryCarrier(info) {
		return parseRecords(info, info.entrySec), nil
	}
	index, err := loadSymbolIndex(path, info)
	if err != nil {
		return nil, err
	}
	var records []siteRecord
	for _, sym := range info.syms {
		id := fnv64(sourceSymbolName(info, sym.name))
		if _, ok := lookupSymIndex(index, id); ok {
			records = append(records, siteRecord{pc: sym.addr, symbolID: id})
		}
	}
	return records, nil
}
