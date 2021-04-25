package lodb_test

import (
  "strings"
  "testing"
  "time"

  "lfm_lookout/internal/lodb"

  badger "github.com/dgraph-io/badger/v3"
)


func TestLoDB_SingleQueryOps(t *testing.T) {
  query := lodb.LoQuery{AuthorID: "xHdx", ChannelID: "hwes", TTL: time.Hour, Query: `Server:Cannith +raid`}

  opt := badger.DefaultOptions("").WithInMemory(true)
  db, err := badger.Open(opt)
  if err != nil {
    t.Fatalf("Error occured while initializing repo: %s", err.Error())
  }
  lorepo := LoRepo{db: db}
  defer LoRepo.Close()

  // Save query one. [0]
  if err := lorepo.Save(query); err != nil {
    t.Fatalf("Error occured while saving first query: %s", err.Error())
  }
  // Retrieve author queries, and ensure there is only one which matches the first one added.
  findSingle, err := lorepo.FindByAuthor(query.AuthorID)
  if err != nil {
    t.Errorf("Error occured while finding query: %s", err.Error())
  } else if len(findSingle) != 1 {
    t.Errorf("Returned slice is of size %d, not 1.", len(findSingle))
  } else if !QueryEqual(query, findSingle[0]) {
    t.Error("Returned query is not the same as the original.")
  }
  // Delete entry. []
  if err := lorepo.Delete('0'); err != nil {
    t.Errorf("Error occured while deleting first query: %s", err.Error())
  }
  // Retrieve author queries, ensure the return slice is empty.
  findEmpty, err := lorepo.FindByAuthor(query.AuthorID)
  if err != nil {
    t.Errorf("Error occured while finding deleted query: %s", err.Error())
  }
  if len(findEmpty) != 0 {
    t.Errorf("Returned slice is of size %d, not 0.", len(findEmpty))
  }
}

func TestLoDB_SingleQueryTTL(t *testing.T) {
  ttl := time.Second
  query := lodb.LoQuery{AuthorID: "xHdx", ChannelID: "hwes", TTL: ttl, Query: `Server:Cannith +raid`}

  opt := badger.DefaultOptions("").WithInMemory(true)
  db, err := badger.Open(opt)
  if err != nil {
    t.Fatalf("Error occured while initializing repo: %s", err.Error())
  }
  lorepo := LoRepo{db: db}
  defer LoRepo.Close()

  // Save query.
  if err := lorepo.Save(query); err != nil {
    t.Fatalf("Error occured while saving first query: %s", err.Error())
  }
  // Sleep for twice the TTL.
  time.Sleep(ttl)
  time.Sleep(ttl)
  // Retrieve author queries, ensure the return slice is empty.
  findEmpty, err := lorepo.FindByAuthor(query.AuthorID)
  if err != nil {
    t.Errorf("Error occured while finding deleted query: %s", err.Error())
  }
  if len(findEmpty) != 0 {
    t.Errorf("Returned slice is of size %d, not 0. Query has not expired.", len(findEmpty))
  }
}

func TestLoDB_IndexRecycling(t *testing.T) {
  queries := []lodb.LoQuery {
    {AuthorID: "xHdx", ChannelID: "hwes", TTL: time.Hour, Query: `Server:Cannith +raid`},
    {AuthorID: "xHdx", ChannelID: "retO", TTL: 5 * time.Second, Query: `Server:Farwarged R1 "Curse of Strahd"`},
    {AuthorID: "xHdx", ChannelID: "qEud", TTL: time.Hour, Query: `Server:Wayfinder feywild`},
    {AuthorID: "xHdx", ChannelID: "qEud", TTL: time.Hour, Query: `Server:Khyber dailies`},
  }

  opt := badger.DefaultOptions("").WithInMemory(true)
  db, err := badger.Open(opt)
  if err != nil {
    t.Fatalf("Error occured while initializing repo: %s", err.Error())
  }
  lorepo := LoRepo{db: db}
  defer LoRepo.Close()

  // Save query one, two, and three. [0,1,2]
  if err := lorepo.Save(queries[0]); err != nil {
    t.Fatalf("Error occured while saving first query: %s", err.Error())
  }
  if err := lorepo.Save(queries[1]); err != nil {
    t.Fatalf("Error occured while saving second query: %s", err.Error())
  }
  if err := lorepo.Save(queries[2]); err != nil {
    t.Fatalf("Error occured while saving third query: %s", err.Error())
  }
  // Retrieve, ensure three entries for author.
  findThree, err := lorepo.FindByAuthor(query.AuthorID)
  if err != nil {
    t.Errorf("Error occured while finding deleted query: %s", err.Error())
  }
  if len(findThree) != 3 {
    t.Errorf("Returned slice is of size %d, not 0. Query has not expired.", len(findThree))
  }
  // Delete third query.
  if err := lorepo.Delete(rune('0')); err != nil {
    t.Error("Error occured while deleting first query from repo.")
  }
  // Retrieve queries, ensure two remain and three is no longer accessible.
  findEmpty, err := lorepo.FindByAuthor(query.AuthorID)
  if err != nil {
    t.Errorf("Error occured while retrieving queries: %s", err.Error())
  }
  if len(findEmpty) != 2 {
    t.Errorf("Returned slice is of size %d, not 2. Query has not expired.", len(findEmpty))
  }
  // Save query four.
  // Retrieve, ensure id=2 was freed and that query four is id=2.

  // Directly add entries with malformed keys, and ensure errors are returned.
}

func QueryEqual(a, b lodb.LoQuery) bool {
  switch {
  case a.AuthorID != b.AuthorID:
    return false
  case a.ChannelID != b.ChannelID:
    return false
  case a.Query != b.Query:
    return false
  default:
    return true
  }
}
