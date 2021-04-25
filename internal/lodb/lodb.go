package lodb

import(
  "errors"
  "fmt"
  "strconv"
  "strings"
  "time"
  "unicode/utf8"

  badger "github.com/dgraph-io/badger/v3"
)

var (
  ErrOrphanPair = errors.New("Key-value entry has no corresponding query or return entry.")
  ErrUserIndicesFull = errors.New("No open index found for user.")
  ErrMalformedKey = errors.New("The key appears malformed and cannot be processed.")
  ErrCorruptQuery = errors.New("A part of the query is corrupted.")
)

const (
  // The minimum value a query's user-unique id can be.
  IDMIN rune = '0'
  // The maximum value a query's user-uniqe id can be.
  IDMAX rune = '9'
  // The maximum life-time of a query.
  TTLMAX time.Duration = time.Hour * 24
)

type LoQuery struct {
  AuthorID string
  ChannelID string
  ID rune // ID is a unique-per-user integer, 0-9.
  TTL time.Duration
  Query string // String formatted for string query in Bleve
}


type LoRepo struct {
  db *badger.DB
}

func NewLoRepo(path string) (*LoRepo, error) {
  db, err := badger.Open(badger.DefaultOptions(path))
  if err != nil {
    return &LoRepo{db: db}, err
  }
  return &LoRepo{db: db}, nil
}

// Close
func (r *LoRepo) Close() {
  r.db.Close()
}

// Delete
func (r *LoRepo) Delete(authorID string, index rune) error {
  // TODO: Should try doing this as a WriteBatch, instead.
  err := r.db.Update(func(txn *badger.Txn) error {
    qKey := fmt.Sprintf("query-%s-%s", authorID, string(index))
    rKey := fmt.Sprintf("return-%s-%s", authorID, string(index))

    err1 := txn.Delete([]byte(qKey))
    if err1 != nil {
      return err1
    }
    err2 := txn.Delete([]byte(rKey))
    if err2 != nil {
      return err2
    }
    return nil
  })
  if err != nil {
    return err
  }
  return nil
}

// // FindByID
// func (r *LoRepo) FindByID(id int) (*LoQuery, error) {
//   var q LoQuery
//   err := r.findByIDStmt.QueryRow(id).Scan(
//     &q.AuthorID,
//     &q.Classes,
//     &q.ChannelID,
//     &q.Duration,
//     &q.ID,
//     &q.Level,
//     &q.Onetime,
//     &q.Timestamp,
//     &q.Query)
//   return &q, err
// }

// FindByAuthorID
func (r *LoRepo) FindByAuthor(authorid string) ([]LoQuery, error) {
  var queries []LoQuery

  err := r.db.View(func(txn *badger.Txn) error {
    it := txn.NewIterator(badger.DefaultIteratorOptions)
    defer it.Close()
    prefix := []byte(fmt.Sprintf("query-%s", authorid))
    for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
      qItem := it.Item()
      var valCopy []byte
      keyStr := string(qItem.Key())
      err := qItem.Value(func(v []byte) error {
        valCopy = append([]byte{}, v...)
        return nil
      })
      if err != nil {
        return err
      }
      if len(keyStr) <= len("query-") + len("-0") {
          return ErrMalformedKey
      }
      // Parse out the AuthorID.
      auth := keyStr[len("query-"):len(keyStr)-len("-0")]

      id, _ := utf8.DecodeLastRuneInString(keyStr)
      if id < IDMIN || id > IDMAX {
        return ErrMalformedKey
      }
      // Retrieve the current TTL.
      expTime := time.Unix(int64(qItem.ExpiresAt()), 0)
      ttl := expTime.Sub(time.Now())
      // TODO: check for valid ttl (not max or min, is less than 24 hours, ect.)
      if ttl > TTLMAX {
        return ErrCorruptQuery
      }

      loQ := LoQuery{
        AuthorID: auth,
        ChannelID: "",
        ID: id,
        TTL: ttl,
        Query: string(valCopy),
      }
      queries = append(queries, loQ)
    }
    return nil
  })
  if err != nil {
    return queries, err
  }
  return queries, nil
}

// Save // TODO: Change to return ID?
func (r *LoRepo) Save(q LoQuery) error {
  // Find lowest unused index.
  err := r.db.Update(func(txn *badger.Txn) error {
    itOpts := badger.DefaultIteratorOptions
    itOpts.PrefetchValues = false
    it := txn.NewIterator(itOpts)
    defer it.Close()
    prefix := []byte("query-"+q.AuthorID)
    unusedIndex := IDMIN
    for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
      item := it.Item()
      k := item.Key()
      curr, _ := utf8.DecodeLastRuneInString(string(k))
      if unusedIndex < curr {
        break
      } else {
        unusedIndex = curr + 1
      }
    }
    if unusedIndex > IDMAX {
      return ErrUserIndicesFull
    }
    // Set unused index to new query.
    // query-[AuthorID]-[0-9]:[Query]
    qKey := fmt.Sprintf("query-%s-%s", q.AuthorID, string(unusedIndex))
    e1 := badger.NewEntry([]byte(qKey), []byte(q.Query)).WithTTL(q.TTL)
    _ = txn.SetEntry(e1)
    // return-[AuthorID]-[0-9]:[ChannelID]
    rKey := fmt.Sprintf("return-%s-%s", q.AuthorID, string(unusedIndex))
    e2 := badger.NewEntry([]byte(rKey), []byte(q.ChannelID)).WithTTL(q.TTL)
    _ = txn.SetEntry(e2)

    return nil
  })
  if err != nil {
    return err
  }
  return nil
}

func (r *LoRepo) GetView(fn func(txn *badger.Txn) error) error {
  return r.db.View(fn)
}

func (q LoQuery) String() string {
  qStrings := []string{}
  qStrings = append(qStrings, "ID: " + strconv.FormatInt(int64(q.ID), 10))

  qStrings = append(qStrings, "Query: \"" + q.Query + "\"")

  qStrings = append(qStrings, "Duration: " + q.TTL.String())

  return strings.Join(qStrings, ", ")
}
