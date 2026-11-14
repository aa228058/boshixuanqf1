package handler

import "testing"

// 7z l -slt 的真实输出片段(节选自 p7zip):
// 每个条目以 "Path = " 开始, 目录条目的 Attributes 以 D 开头, Packed Size 不能误判为 Size。
const sampleListing = `7-Zip (a) [64] 16.02 : Copyright (c) 1999-2023 Igor Pavlov

Listing archive: sample.7z

--
Path = sample.7z
Type = 7z
Physical Size = 4096
Headers Size = 200

----------
Path = accounts/
Size = 0
Packed Size = 0
Attributes = D_ drwxr-xr-x
CRC =
Encrypted = -

Path = accounts/8613800000000.session
Size = 204800
Packed Size = 1024
Attributes = A_ -rw-r--r--
CRC = 12345678
Encrypted = -

Path = accounts/8613800000000.json
Size = 4096
Packed Size = 512
Attributes = A_ -rw-r--r--
CRC = 87654321
Encrypted = -

`

func TestParseArchiveListingCountsFilesAndSkipsDirectories(t *testing.T) {
	entries, total := parseArchiveListing(sampleListing)
	if entries != 2 {
		t.Fatalf("entries = %d, want 2 (目录不应计入)", entries)
	}
	const want = int64(204800 + 4096)
	if total != want {
		t.Fatalf("total = %d, want %d", total, want)
	}
}

func TestParseArchiveListingIgnoresPackedSize(t *testing.T) {
	// Packed Size 是压缩后体积, 若被当成 Size 会低估解压后大小, 让压缩炸弹通过预检。
	const listing = `Path = a.session
Size = 1048576
Packed Size = 8
Attributes = A_ -rw-r--r--
`
	entries, total := parseArchiveListing(listing)
	if entries != 1 {
		t.Fatalf("entries = %d, want 1", entries)
	}
	if total != 1048576 {
		t.Fatalf("total = %d, want 1048576", total)
	}
}

func TestParseArchiveListingEmpty(t *testing.T) {
	entries, total := parseArchiveListing("")
	if entries != 0 || total != 0 {
		t.Fatalf("空清单应得到 0/0, 实际 %d/%d", entries, total)
	}
}
