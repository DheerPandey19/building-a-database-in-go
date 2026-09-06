# Building a Database in Go

A persistent key-value store built from scratch in Go, following the
[Build Your Own Database](https://build-your-own.org/) style of learning: start
with on-disk pages, then a copy-on-write B+tree, durability via `fsync`, and a
free list for page reuse.

## What this project implements

| Layer | What it does |
|-------|----------------|
| **B+tree** | In-memory tree over 4KB nodes (`BNode`), with insert, lookup, and node splits |
| **KV store** | `Open`, `Get`, `Set` over a single database file |
| **Pages** | Append new pages or reuse freed ones; pending writes live in an `updates` map |
| **Free list** | Linked list of free page numbers (`FreeList` / `LNode`) so COW can recycle pages |
| **Meta page** | Atomic root + free-list pointers (`head`/`tail` seq) written after data pages |
| **Durability** | Write pages → `fsync` → update meta → `fsync` |

### File layout

```text
Page 0  — meta: signature | root | page_used | free head/tail
Page 1  — initial free-list node (empty DB always has ≥1 free-list node)
Page 2+ — B+tree nodes and additional free-list nodes
```

### Public API (current)

```go
db := &KV{Path: "test.db"}
db.Open()
db.Set([]byte("key"), []byte("value"))
val := db.Get([]byte("key"))
```

`Set` is durable (pages + meta are synced). Deletes at the tree level free pages
into the free list via COW; a high-level `Del` API is not wired in `main` yet.

## Requirements

- **Go 1.18+**
- **Unix / Linux** (or WSL on Windows): uses `syscall.Mmap`, `Pwrite`, `Fsync`

Native Windows `go build` will fail on those syscalls. Use WSL.

## Run

From the project directory in WSL:

```bash
cd /mnt/c/Users/<you>/.../BuildingADatabase
go run .
```

Or from PowerShell:

```powershell
wsl -d Ubuntu -- bash -lc 'cd /mnt/c/Users/<you>/.../BuildingADatabase && go run .'
```

`main.go` deletes `test.db`, writes a few keys, reopens the file, checks
persistence, updates a key, and reopens again.

Expected output:

```text
Written values:
name = Dheer
language = Go

After reopening:
OK: name = Dheer
OK: language = Go
OK: project = build-your-own-db

All persistence tests passed.
```

## Project layout

```text
btree.go   — B+tree, free list, page manager, KV Open/Get/Set
main.go    — persistence smoke test
go.mod     — module build-your-own-db
```

## Design notes

- **Copy-on-write**: updates allocate new pages; old pages go to the free list.
- **Free list safety**: `maxSeq` snapshots `tailSeq` so a transaction cannot
  reuse pages it just freed in the same update.
- **Rollback**: if a flush fails mid-update, in-memory meta is restored and
  pending `updates` are discarded (`failed` + `updateOrRevert`).

## Not in this repo (typical next steps)

- Concurrent readers / MVCC
- Relational layer on top of the KV store
- Windows-native page I/O (no Unix mmap)


